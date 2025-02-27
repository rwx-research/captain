package main

import (
	_ "embed"
	"encoding/json"

	"github.com/rwx-research/captain-cli/internal/errors"
	v1 "github.com/rwx-research/captain-cli/internal/testingschema/v1"
)

type IdentityRecipe struct {
	Language string
	Kind     string
	Recipe   struct {
		Components []string
		Strict     bool
	}
}

//go:embed identity_recipes.json
var recipeJSON []byte

func getRecipes() (map[string]v1.TestIdentityRecipe, error) {
	var recipeList []IdentityRecipe
	recipes := make(map[string]v1.TestIdentityRecipe)

	if err := json.Unmarshal(recipeJSON, &recipeList); err != nil {
		return recipes, errors.NewInternalError("unable to parse identiy recipes: %s", err.Error())
	}

	for _, identityRecipe := range recipeList {
		recipes[v1.CoerceFramework(identityRecipe.Language, identityRecipe.Kind).String()] = v1.TestIdentityRecipe{
			Components: identityRecipe.Recipe.Components,
			Strict:     identityRecipe.Recipe.Strict,
		}
	}

	return recipes, nil
}
