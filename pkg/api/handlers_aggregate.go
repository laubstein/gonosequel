package api

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/laubstein/gonosequel/pkg/client"
)

// handleAggregate runs an aggregation pipeline and returns the resulting
// documents as relaxed Extended JSON.
//
// This is a POST, and --readonly's middleware rejects every non-GET
// request — deliberately here, not just incidentally: unlike a find
// filter, a pipeline can itself write via $out or $merge, and there is no
// cheap, reliable way to tell a read-only pipeline from one that isn't
// short of inspecting every stage, so --readonly blocks aggregate
// entirely rather than trying to allow "safe" pipelines through.
func (d *deps) handleAggregate(c fiber.Ctx) error {
	db, coll := c.Params("db"), c.Params("coll")

	pipeline, err := client.FromExtJSONArray(c.Body())
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid pipeline: "+err.Error())
	}

	docs, err := currentClient(c).Aggregate(c.Context(), db, coll, pipeline)
	if err != nil {
		return fmt.Errorf("aggregate: %w", err)
	}

	wrapper := bson.M{"documents": docs}
	body, err := client.ToRelaxedExtJSON(wrapper)
	if err != nil {
		return fmt.Errorf("marshal aggregate results: %w", err)
	}
	return c.Type("json").Send(body)
}
