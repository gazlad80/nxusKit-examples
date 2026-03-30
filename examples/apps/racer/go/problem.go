// Package racer provides problem loading functionality.
package racer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ProblemLoader loads problems from various sources.
type ProblemLoader struct {
	// Registry is the problem registry to populate.
	Registry *ProblemRegistry
	// DataDir is the base directory for problem data files.
	DataDir string
}

// NewProblemLoader creates a new problem loader.
func NewProblemLoader(dataDir string) *ProblemLoader {
	return &ProblemLoader{
		Registry: NewProblemRegistry(),
		DataDir:  dataDir,
	}
}

// LoadFromFile loads a problem from a JSON file.
func (l *ProblemLoader) LoadFromFile(path string) (*Problem, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read problem file: %w", err)
	}

	var problem Problem
	if err := json.Unmarshal(data, &problem); err != nil {
		return nil, fmt.Errorf("failed to parse problem JSON: %w", err)
	}

	l.Registry.Register(&problem)
	return &problem, nil
}

// LoadFromDirectory loads all problems from a directory.
func (l *ProblemLoader) LoadFromDirectory(dir string) ([]*Problem, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var problems []*Problem
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		problem, err := l.LoadFromFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", entry.Name(), err)
		}
		problems = append(problems, problem)
	}

	return problems, nil
}

// LoadBuiltinProblems loads the built-in problem set.
func (l *ProblemLoader) LoadBuiltinProblems() error {
	if l.DataDir == "" {
		return fmt.Errorf("data directory not set")
	}

	_, err := l.LoadFromDirectory(l.DataDir)
	return err
}

// GetBuiltinProblemRegistry returns a registry with built-in problems.
func GetBuiltinProblemRegistry() *ProblemRegistry {
	registry := NewProblemRegistry()

	// Register built-in problems
	registry.Register(NewEinsteinRiddle())
	registry.Register(NewFamilyRelations())
	registry.Register(NewAnimalClassification())

	return registry
}

// NewEinsteinRiddle creates the Einstein riddle problem.
func NewEinsteinRiddle() *Problem {
	inputData := json.RawMessage(`{
		"houses": 5,
		"attributes": ["color", "nationality", "drink", "cigarette", "pet"],
		"clues": [
			"The Brit lives in the red house",
			"The Swede keeps dogs as pets",
			"The Dane drinks tea",
			"The green house is on the left of the white house",
			"The green house's owner drinks coffee",
			"The person who smokes Pall Mall rears birds",
			"The owner of the yellow house smokes Dunhill",
			"The man living in the center house drinks milk",
			"The Norwegian lives in the first house",
			"The man who smokes blends lives next to the one who keeps cats",
			"The man who keeps horses lives next to the man who smokes Dunhill",
			"The owner who smokes BlueMaster drinks beer",
			"The German smokes Prince",
			"The Norwegian lives next to the blue house",
			"The man who smokes blend has a neighbor who drinks water"
		],
		"question": "Who owns the fish?"
	}`)

	expectedSolution := json.RawMessage(`{"fish_owner": "German"}`)

	return NewProblem("einstein-riddle", ProblemTypeLogicPuzzle,
		"The classic Einstein riddle: Given 15 clues about 5 houses, determine who owns the fish.").
		WithInput(inputData).
		WithSolution(expectedSolution).
		WithDifficulty(DifficultyHard)
}

// NewFamilyRelations creates the family relations problem.
func NewFamilyRelations() *Problem {
	inputData := json.RawMessage(`{
		"people": [
			{"name": "Alice", "gender": "female"},
			{"name": "Bob", "gender": "male"},
			{"name": "Carol", "gender": "female"},
			{"name": "David", "gender": "male"},
			{"name": "Eve", "gender": "female"},
			{"name": "Frank", "gender": "male"},
			{"name": "Grace", "gender": "female"},
			{"name": "Henry", "gender": "male"}
		],
		"relationships": [
			{"type": "parent", "parent": "Alice", "child": "Carol"},
			{"type": "parent", "parent": "Alice", "child": "David"},
			{"type": "parent", "parent": "Bob", "child": "Carol"},
			{"type": "parent", "parent": "Bob", "child": "David"},
			{"type": "parent", "parent": "Carol", "child": "Eve"},
			{"type": "parent", "parent": "Carol", "child": "Frank"},
			{"type": "parent", "parent": "David", "child": "Grace"},
			{"type": "parent", "parent": "David", "child": "Henry"}
		],
		"queries": [
			"Who are Alice's grandchildren?",
			"Are Eve and Grace cousins?",
			"Who is David's uncle or aunt?"
		]
	}`)

	expectedSolution := json.RawMessage(`{"relationships_found": true}`)

	return NewProblem("family-relations", ProblemTypeLogicPuzzle,
		"Infer family relationships: grandparents, cousins, uncles, aunts from parent-child facts.").
		WithInput(inputData).
		WithSolution(expectedSolution).
		WithDifficulty(DifficultyMedium)
}

// NewAnimalClassification creates the animal classification problem.
func NewAnimalClassification() *Problem {
	inputData := json.RawMessage(`{
		"animals": [
			{"name": "dog", "has_fur": true, "warm_blooded": true, "gives_milk": true},
			{"name": "eagle", "has_feathers": true, "warm_blooded": true, "can_fly": true},
			{"name": "snake", "has_scales": true, "cold_blooded": true, "lays_eggs": true},
			{"name": "frog", "moist_skin": true, "metamorphosis": true, "cold_blooded": true},
			{"name": "salmon", "has_gills": true, "has_fins": true, "cold_blooded": true},
			{"name": "whale", "warm_blooded": true, "gives_milk": true, "lives_in_water": true},
			{"name": "penguin", "has_feathers": true, "warm_blooded": true, "cannot_fly": true},
			{"name": "platypus", "has_fur": true, "gives_milk": true, "lays_eggs": true}
		],
		"classes": ["mammal", "bird", "reptile", "amphibian", "fish"]
	}`)

	expectedSolution := json.RawMessage(`{
		"classifications": [
			{"animal": "dog", "class": "mammal"},
			{"animal": "eagle", "class": "bird"},
			{"animal": "snake", "class": "reptile"},
			{"animal": "frog", "class": "amphibian"},
			{"animal": "salmon", "class": "fish"},
			{"animal": "whale", "class": "mammal"},
			{"animal": "penguin", "class": "bird"},
			{"animal": "platypus", "class": "mammal"}
		]
	}`)

	return NewProblem("animal-classification", ProblemTypeClassification,
		"Classify animals into their taxonomic classes based on observable characteristics.").
		WithInput(inputData).
		WithSolution(expectedSolution).
		WithDifficulty(DifficultyEasy)
}

// FindProblemByName searches for a problem by name with fuzzy matching.
func FindProblemByName(registry *ProblemRegistry, name string) (*Problem, []string) {
	// Exact match first
	if problem := registry.Get(name); problem != nil {
		return problem, nil
	}

	// Collect similar names for suggestions
	var suggestions []string
	for _, registeredName := range registry.List() {
		if stringSimilarity(name, registeredName) > 0.5 {
			suggestions = append(suggestions, registeredName)
		}
	}

	return nil, suggestions
}

// stringSimilarity calculates a simple similarity score between two strings.
// Returns a value between 0 and 1.
func stringSimilarity(a, b string) float64 {
	if a == b {
		return 1.0
	}
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	// Simple Jaccard similarity on characters
	aChars := make(map[rune]bool)
	for _, c := range a {
		aChars[c] = true
	}

	bChars := make(map[rune]bool)
	for _, c := range b {
		bChars[c] = true
	}

	intersection := 0
	for c := range aChars {
		if bChars[c] {
			intersection++
		}
	}

	union := len(aChars) + len(bChars) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}
