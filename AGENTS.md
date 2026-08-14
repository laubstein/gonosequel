# AGENTS.md

Instruções para agentes que trabalham neste repositório. O racional, o escopo e as decisões de
produto estão em [`PLAN.md`](./PLAN.md) — este arquivo é normativo e diz o que fazer e o que não
fazer.

## O que é

`Go NoSequel` é um explorador web de MongoDB em Go: a interface e a usabilidade do
[pgweb](https://github.com/sosedoff/pgweb) com as funcionalidades do
[mongo-express](https://github.com/mongo-express/mongo-express). Compila para um binário único com
o frontend embutido.

**Estado atual: projeto do zero.** Nada foi implementado ainda. Siga a "Ordem de execução" do
`PLAN.md` — o `pkg/client/extjson.go` e seus testes vêm antes de qualquer outra coisa, porque são a
base de correção do resto.

## Comandos

```bash
make build       # cd web && npm run build  →  go build -o gonosequel .
make test        # unidade + integração; sobe e derruba o Mongo em Docker sozinho
make test-short  # só unidade (go test ./... -short); não precisa de Docker
make lint        # gofmt -l . && go vet ./... && staticcheck ./... && errcheck ./...
make dev         # sobe Mongo (Docker se preciso), API Go e Vite; abra a URL do Vite
make dev-down    # remove o container Mongo que `make dev` possa ter criado
```

Rode `make lint` e `make test` antes de dar qualquer trabalho por concluído. Se o Docker não
estiver disponível no ambiente, rode `make test-short` e **diga explicitamente** que os testes de
integração não foram executados — não relate a suíte como verde.

## Dependências

Fixadas nestas versões; não faça downgrade nem troque sem justificar.

| Módulo | Versão |
|---|---|
| `github.com/gofiber/fiber/v3` | v3.5.0 |
| `go.mongodb.org/mongo-driver/v2` | v2.8.0 |
| `github.com/testcontainers/testcontainers-go/modules/mongodb` | v0.44.0 |
| `github.com/BurntSushi/toml` | v1.x |

Go 1.26+, Node 24+. O backend não deve ganhar dependências além dessas sem uma boa razão — o
objetivo é binário único e enxuto.

## Estrutura

```
main.go              flags, bootstrap, ciclo de vida — e nada mais
pkg/command/         flags CLI + env vars
pkg/bookmarks/       ~/.gonosequel/bookmarks/*.toml
pkg/client/          toda a interação com o MongoDB
pkg/session/         registro de sessões (modo --sessions)
pkg/history/         histórico de queries, em memória, por sessão
pkg/export/          JSON e CSV em streaming
pkg/api/             *fiber.App: server.go, routes.go, handlers_*.go, middleware.go, assets.go
web/                 React 19 + TypeScript + Vite
```

`pkg/client` não importa `pkg/api` nem conhece HTTP. `pkg/api` não fala com o driver do Mongo
diretamente.

## Regras invioláveis

Estas quatro causam corrupção de dados, vazamento de recursos ou falha de segurança quando
violadas. Não as contorne.

**1. Documentos trafegam como Extended JSON, nunca como JSON comum.**
BSON não mapeia para JSON sem perda. Use `bson.MarshalExtJSON` / `bson.UnmarshalExtJSON`.
Backend → frontend em **relaxed** (legível); ao servir um documento para **edição**, em
**canonical**, para que o round-trip ver→editar→salvar não converta um `Long` em `Double` em
silêncio. Nos handlers, devolva o extended JSON já serializado com `c.Type("json").Send(raw)` —
**nunca** `c.JSON(...)`, que aplicaria um segundo passe de marshaling e destruiria os tipos.

**2. `_id` pode ser qualquer tipo BSON.**
Nunca assuma ObjectID hex. O path param de documento é o base64url do extended JSON canonical do
`_id`; use `client.EncodeDocID` / `client.DecodeDocID`.

**3. `--readonly` é imposto no middleware.**
Rejeite todo método não-GET com 403 no servidor. Esconder botões na UI é sugestão, não trava.

**4. Propague o contexto da requisição.**
Toda chamada ao driver recebe o `c.Context()` do handler, para que uma query cara seja cancelada
quando o cliente desconecta. `context.Context` é sempre o primeiro parâmetro e nunca é guardado
em struct. `defer` para fechar cursores e conexões.

Além dessas: na paginação, use `EstimatedDocumentCount` (O(1)) quando não há filtro e
`CountDocuments` só quando há — contar dezenas de milhões de documentos a cada página trava a UI.
No export, escreva direto do cursor com `c.SendStreamWriter`, sem materializar o resultado.

## Convenções Go (Effective Go)

- **Nomes**: pacotes curtos, minúsculos, sem underscores. Sem stutter: `client.New()`, não
  `client.NewClient()`. Getters sem prefixo `Get`.
- **Erros**: `fmt.Errorf` com `%w` para envolver; sentinelas exportadas (`ErrNotFound`,
  `ErrReadOnly`) verificadas com `errors.Is`. Mensagens em minúscula, sem pontuação final.
  Sem `panic` fora de erro de programação em `init`.
- **Interfaces** definidas onde são **consumidas**, não onde são implementadas: `pkg/api` declara
  a interface estreita de que precisa; `pkg/client` devolve tipos concretos. Interfaces pequenas
  (1–3 métodos) — isso também é o que torna os handlers testáveis com fakes.
- **Concorrência**: o registro de sessões protege seu mapa com `sync.RWMutex`.
- **Documentação**: todo identificador exportado tem doc comment começando pelo próprio nome.
- Handlers Fiber v3 são `func(c fiber.Ctx) error` — `Ctx` é interface **por valor** na v3, não
  ponteiro como na v2. Erros vão para o `ErrorHandler` central; nenhum handler escreve corpo de
  erro à mão.

## Testes

Três camadas, todas em `make test`:

1. **Unidade, sem Docker** — table-driven, com `t.Parallel()`. O round-trip de cada tipo BSON
   (`ObjectId`, `ISODate`, `Decimal128`, `Long`, `Binary`, `DBRef`, `MinKey`/`MaxKey`) é a suíte
   mais importante do projeto: é onde uma regressão corrompe dados do usuário em silêncio.
2. **Integração contra Mongo real** — `testcontainers-go/modules/mongodb` sobe um `mongo:8` no
   `TestMain`. Não exija que o usuário rode `docker run` antes. Cada teste usa um banco de nome
   único e o remove no `t.Cleanup`, para rodarem em paralelo sem interferência.
3. **HTTP ponta a ponta** — `app.Test(req)` do Fiber contra o Mongo do container; sem porta aberta,
   sem servidor real.

Testes que precisam de Docker pulam com `testing.Short()`.

## Escopo do v1 — não implemente

Fora do escopo por decisão explícita. Não adicione sem pedir:
GridFS, pipeline de aggregation, gestão de usuários/roles, preview inline de mídia, túnel SSH.

## Frontend

React 19 + TypeScript + Vite. TanStack Query para estado de servidor (cache + invalidação em
mutações). CSS Modules com custom properties — tema claro/escuro sem framework de CSS.

O layout é o do pgweb, três painéis: abas no topo; sidebar com seletor de banco, filtro de coleções
e painel de estatísticas; corpo com editor em cima, resultados no meio, filtro e paginação no rodapé.
A diferença essencial em relação ao pgweb é o **alternador tabela/JSON** nos resultados: documentos
aninhados não cabem em uma tabela relacional, então a visão JSON com nós recolhíveis não é opcional.
