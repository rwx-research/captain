package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"go.uber.org/zap"

	"github.com/rwx-research/captain-cli"
	"github.com/rwx-research/captain-cli/internal/backend/remote"
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

type Cache struct {
	CaptainVersion string
	Recipes        []IdentityRecipe
}

func getRecipes(logger *zap.SugaredLogger, cfg Config) (map[string]v1.TestIdentityRecipe, error) {
	var cache Cache

	existingCaptainDir, err := findInParentDir(captainDirectory)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}

		if err = os.Mkdir(captainDirectory, 0o755); err != nil {
			return nil, errors.WithStack(err)
		}

		existingCaptainDir = captainDirectory
	}

	recipesFile, err := findInParentDir(filepath.Join(captainDirectory, "recipes.json"))
	if err == nil {
		// Fall back to getting recipes from Cloud on error
		cache, err = getRecipesFromCache(recipesFile)
	}
	if err != nil {
		cache, err = getRecipesFromCloud(logger, cfg)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		if recipesFile == "" {
			recipesFile = filepath.Join(existingCaptainDir, "recipes.json")
		}

		// Write to cache file
		buffer, err := json.Marshal(cache)
		if err == nil {
			err = os.WriteFile(recipesFile, buffer, 0o600)
		}
		if err != nil {
			logger.Warnf("unable to cache identity recipes on disk: %s", err.Error())
		}
	}

	recipes := make(map[string]v1.TestIdentityRecipe)
	for _, identityRecipe := range cache.Recipes {
		recipes[v1.CoerceFramework(identityRecipe.Language, identityRecipe.Kind).String()] = v1.TestIdentityRecipe{
			Components: identityRecipe.Recipe.Components,
			Strict:     identityRecipe.Recipe.Strict,
		}
	}

	return recipes, nil
}

func getRecipesFromCache(filePath string) (Cache, error) {
	var buffer []byte
	var cache Cache

	buffer, err := os.ReadFile(filePath)
	if err != nil {
		return Cache{}, errors.WithStack(err)
	}

	if err = json.Unmarshal(buffer, &cache); err != nil {
		return Cache{}, errors.WithStack(err)
	}

	if cache.CaptainVersion != captain.Version {
		return Cache{}, errors.NewCacheError("Outdated Captain Version")
	}

	return cache, nil
}

func getRecipesFromCloud(logger *zap.SugaredLogger, cfg Config) (Cache, error) {
	var buffer []byte
	var recipeList []IdentityRecipe

	client, err := remote.NewClient(remote.ClientConfig{
		Debug:    cfg.Output.Debug,
		Host:     cfg.Cloud.APIHost,
		Insecure: cfg.Cloud.Insecure,
		Log:      logger,
		Token:    "none", // Can't be empty. We rely on implementation details here that `GetIdentityRecipes` will not use it
	})
	if err != nil {
		return Cache{}, errors.Wrap(err, "Unable to initialize API client")
	}

	buffer, err = client.GetIdentityRecipes(context.Background())
	if err != nil {
		return Cache{}, errors.Wrap(err, "Unable to fetch test identity recipes from API")
	}

	if err = json.Unmarshal(buffer, &recipeList); err != nil {
		return Cache{}, errors.NewInternalError("unable to parse identiy recipes: %s", err.Error())
	}

	return Cache{
		CaptainVersion: captain.Version,
		Recipes:        recipeList,
	}, nil
}
