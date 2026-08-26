# AGENTS.md

Instruções para agentes que trabalham neste repositório. Este arquivo é normativo e diz o que
fazer e o que não fazer.

## O que é

`Go NoSequel` é um explorador web de NoSQL em Go: a interface e a usabilidade do
[pgweb](https://github.com/sosedoff/pgweb) com as funcionalidades do
[mongo-express](https://github.com/mongo-express/mongo-express). Compila para um binário único com
o frontend embutido. Conecta a **MongoDB** ou **Redis/Valkey** via `--driver`
(`pkg/driver.Driver` é a interface que abstrai o backend — `pkg/client` implementa para MongoDB,
`pkg/redis` para Redis/Valkey).

**Estado atual: MongoDB e Redis/Valkey implementados e em uso.**

## Comandos

```bash
make build       # cd web && npm run build  →  go build -o gonosequel .
make test        # unidade + integração; sobe e derruba Mongo e Redis em Docker sozinho
make test-short  # só unidade (go test ./... -short); não precisa de Docker
make lint        # gofmt -l . && go vet ./... && staticcheck ./... && errcheck ./...
make dev         # sobe Mongo (Docker se preciso), API Go e Vite; abra a URL do Vite — só MongoDB
make dev-down    # remove o container Mongo que `make dev` possa ter criado
```

`make dev` é específico de MongoDB (é para iterar na aplicação em si). Para desenvolver contra
Redis/Valkey, suba um servidor à parte e rode o binário com `--driver redis --url redis://...`.

Rode `make lint` e `make test` antes de dar qualquer trabalho por concluído. Se o Docker não
estiver disponível no ambiente, rode `make test-short` e **diga explicitamente** que os testes de
integração não foram executados — não relate a suíte como verde.

## Dependências

Fixadas nestas versões; não faça downgrade nem troque sem justificar.

| Módulo | Versão |
|---|---|
| `github.com/gofiber/fiber/v3` | v3.5.0 |
| `go.mongodb.org/mongo-driver/v2` | v2.8.0 |
| `github.com/redis/go-redis/v9` | v9.22.0 |
| `github.com/testcontainers/testcontainers-go/modules/mongodb` | v0.44.0 |
| `github.com/testcontainers/testcontainers-go/modules/redis` | v0.44.0 |
| `github.com/BurntSushi/toml` | v1.x |

Go 1.26+, Node 24+. O backend não deve ganhar dependências além dessas sem uma boa razão — o
objetivo é binário único e enxuto.

## Estrutura

```
main.go              flags, bootstrap, ciclo de vida — e nada mais
pkg/command/         flags CLI + env vars (GNS_*, com fallback ME_*)
pkg/bookmarks/       ~/.gonosequel/bookmarks/*.toml
pkg/driver/          interface Driver que abstrai o backend — nem pkg/client nem pkg/redis
                      são conhecidos fora daqui e de main.go
pkg/client/          implementação de driver.Driver para MongoDB
pkg/redis/           implementação de driver.Driver para Redis/Valkey (mesmo driver p/ ambos)
pkg/session/         registro de sessões (modo --sessions), incl. readonly por sessão
pkg/history/         histórico de queries, em memória, por sessão
pkg/export/          JSON e CSV em streaming
pkg/api/             *fiber.App: server.go, routes.go, handlers_*.go, middleware.go, assets.go
web/                 React 19 + TypeScript + Vite
docs/                VitePress site, embedded at /doc via go:embed (main.go's docsFS)
.github/workflows/   release.yml (binários por tag v*), pages.yml (docs/ no GitHub Pages)
```

`docs/.vitepress/config.ts`'s `base` é lido de `DOCS_BASE` (default `/doc/`, o caminho onde o
binário serve os docs embutidos). Só `pages.yml` sobrescreve para `/<repo>/`, já que um GitHub
Pages de projeto serve a partir de um subcaminho, não da raiz — não hardcode um dos dois valores
sem essa distinção.

Nem `pkg/client` nem `pkg/redis` importam `pkg/api` ou conhecem HTTP. `pkg/api` só fala com
`driver.Driver` — nunca importa `pkg/client`/`pkg/redis` diretamente (só `main.go` faz esse
dispatch, via `Config.Connect`).

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
Existe também readonly por sessão (`session.Info.Readonly`, opcional via a tela de conexão em
modo `--sessions`), verificado em `withSession` — independente do `--readonly` global, mas
nunca menos restritivo: `handleConnect` força a sessão como readonly se o servidor já estiver
em `--readonly`, mesmo que o request de conexão não peça isso.

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
2. **Integração contra servidor real** — `testcontainers-go/modules/mongodb` sobe um `mongo:8`
   e `testcontainers-go/modules/redis` sobe um `redis:8`, ambos no `TestMain` (de `pkg/client`,
   `pkg/redis`, e `pkg/api`). Não exija que o usuário rode `docker run` antes. Cada teste usa um
   banco de nome único e o remove no `t.Cleanup`, para rodarem em paralelo sem interferência.
3. **HTTP ponta a ponta** — `app.Test(req)` do Fiber contra os containers reais; sem porta aberta,
   sem servidor real.

Testes que precisam de Docker pulam com `testing.Short()`.

## Fora de escopo — não implemente

Fora do escopo por decisão explícita. Não adicione sem pedir:
GridFS, gestão de usuários/roles, preview inline de mídia, túnel SSH, CouchDB (planejado, mas
sem trabalho iniciado). Pipeline de aggregation **está implementado** (modo Aggregate do
`QueryEditor`, MongoDB apenas) — não é mais escopo excluído, mencionado aqui só porque a versão
anterior deste arquivo listava errado.

## Frontend

React 19 + TypeScript + Vite. TanStack Query para estado de servidor (cache + invalidação em
mutações). CSS Modules com custom properties — tema claro/escuro sem framework de CSS.

O layout é o do pgweb, três painéis: abas no topo; sidebar com seletor de banco, filtro de coleções
e painel de estatísticas; corpo com editor em cima, resultados no meio, filtro e paginação no rodapé.
A diferença essencial em relação ao pgweb é o **alternador tabela/JSON** nos resultados: documentos
aninhados não cabem em uma tabela relacional, então a visão JSON com nós recolhíveis não é opcional.

`pkg/command/options.go` tem duas camadas de compat com nomes de env var de outro sistema, além
da genérica `GNS_<NAME>`/`ME_<NAME>` (`envLookup`): `envLookupMongo` (condicionada a `--driver
mongodb`, para `MONGODB_HOST`/`MONGODB_PORT`/`MONGODB_USERNAME`/`MONGODB_PASSWORD`) e
`envLookupCompat` (incondicional, para os nomes reais do mongo-express —
`ME_CONFIG_BASICAUTH_USERNAME`/`_PASSWORD`, `ME_CONFIG_SITE_SSL_CRT_PATH`/`_KEY_PATH`,
`ME_CONFIG_SITE_SESSIONSECRET` — em `--auth-user`/`--auth-pass`/`--tls-cert`/`--tls-key`/
`--session-secret`). Ao adicionar uma flag nova que espelhe uma variável real do mongo-express,
prefira estender uma dessas duas em vez de inventar uma terceira camada.

`--auth-enabled`/`--tls-enabled` (default `true`) existem só pra honrar um
`ME_CONFIG_BASICAUTH_ENABLED`/`ME_CONFIG_SITE_SSL_ENABLED=false` importado — no mongo-express
essas variáveis são o interruptor principal (default `false`); no gonosequel a convenção já
é presença de `--auth-user`/`--tls-cert`+`--tls-key` ligar a feature, então essas duas só têm
efeito quando setadas explicitamente para `false` (main.go zera as credenciais/cert antes de
montar `api.Config`/decidir o `Listen`, em vez de repassar um "enabled" pra dentro de
`pkg/api`).

Em `--sessions` mode, o ID de sessão devolvido por `/api/connect` é opaco e sem assinatura por
padrão — trafega no header `X-Session-Id` (`pkg/api/middleware.go`), nunca em cookie. Se
`--session-secret` estiver configurado, `pkg/session/sign.go` (`SignID`/`VerifyID`, HMAC-SHA256)
assina o ID devolvido ao cliente, e `resolveSessionID` (`pkg/api/middleware.go`) passa a exigir
essa assinatura em todo request com `X-Session-Id` — um ID cru ou adulterado vira 400. A chave do
`session.Registry` continua sendo sempre o ID não assinado; só o token que atravessa o header
carrega a assinatura. Sem secret configurado, nada muda (comportamento atual, sem assinatura).

`main.go` sempre imprime um banner no startup (`Options.Banner()`, em `pkg/command/options.go`)
com a config efetiva — driver, alvo de conexão, bind, se sessions/readonly/auth/TLS/session
secret estão ligados. Qualquer campo com cara de credencial (`Pass`, `AuthPass`, `SessionSecret`,
ou a senha embutida numa URI) é mascarado como `****` — nunca logue nenhum desses em texto puro
em lugar nenhum do código; passe por `Banner()`/`redactURI` (a cópia local em
`pkg/command/options.go`, distinta da de `pkg/api/handlers_connect.go` — a mesma ideia, mas cada
uma existe pro seu próprio pacote e propósito) em vez de montar uma string com o segredo cru.
`--verbose` (`Options.Verbose`, e `Config.Verbose`/`deps.verbose` em `pkg/api`) liga linhas de log
extras além do banner (sempre impresso) — hoje só ciclo de vida de sessão em `--sessions` mode
(`handleConnect`/`handleDisconnect` em `pkg/api/handlers_connect.go`). Ao instrumentar um novo
ponto do código, siga o mesmo padrão (`if d.verbose { log.Printf(...) }` / `if opts.Verbose { ... }`)
em vez de introduzir uma abstração de logger nova.

O rascunho do `QueryEditor` e do `RedisCommandRunner` (texto ainda não rodado) é persistido em
`localStorage` (`web/src/api/localCache.ts`), chaveado por `db:coll` — não pelo ID de sessão, de
propósito, para sobreviver tanto a um refresh quanto a uma reconexão (nova sessão) na mesma
collection. O `sessionId` em si (`web/src/api/http.ts`) também é persistido em `localStorage` pelo
mesmo motivo: em modo `--sessions`, perdê-lo a cada refresh derrubava toda chamada com sessão
(sem sessão "default" para cair de volta fora do modo single-connection). Não volte nenhum dos
dois a viver só em memória.

Entrada JSON solta (chaves sem aspas, aspas simples) é detectada via `json5`
(`web/src/api/jsonFix.ts`) e oferece um botão "Fix JSON" no `QueryEditor` e no `DocumentEditor` —
reaproveite esse helper em vez de duplicar a lógica de detecção/correção.
