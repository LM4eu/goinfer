// Copyright 2025 The contributors of Goinfer.
// This file is part of Goinfer, a LLM proxy under the MIT License.
// SPDX-License-Identifier: MIT

package config

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/billziss-gh/golib/shlex"
	"github.com/goccy/go-yaml"
)

const DEFAULT_GROUP_ID = "(default)"

type MacroEntry struct {
	Value any
	Name  string
}

type MacroList []MacroEntry

// UnmarshalYAML implements custom YAML unmarshaling that preserves macro definition order.
func (ml *MacroList) UnmarshalYAML(value []byte) error {
	// Parse the YAML into a yaml.MapSlice to preserve order
	var mapSlice yaml.MapSlice
	if err := yaml.Unmarshal(value, &mapSlice); err != nil {
		return err
	}

	// Clear the existing list
	*ml = MacroList{}

	// Convert the MapSlice to MacroList
	for _, item := range mapSlice {
		key, ok := item.Key.(string)
		if !ok {
			return errors.New("map key is not a string")
		}

		*ml = append(*ml, MacroEntry{
			Name:  key,
			Value: item.Value,
		})
	}

	return nil
}

// Get retrieves a macro value by name.
func (ml MacroList) Get(name string) (any, bool) {
	for _, entry := range ml {
		if entry.Name == name {
			return entry.Value, true
		}
	}
	return nil, false
}

// ToMap converts MacroList to a map (for backward compatibility if needed).
func (ml MacroList) ToMap() map[string]any {
	result := make(map[string]any, len(ml))
	for _, entry := range ml {
		result[entry.Name] = entry.Value
	}
	return result
}

type GroupConfig struct {
	Members    []string `yaml:"members"`
	Swap       bool     `yaml:"swap"`
	Exclusive  bool     `yaml:"exclusive"`
	Persistent bool     `yaml:"persistent"`
}

var (
	macroNameRegex    = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	macroPatternRegex = regexp.MustCompile(`\$\{([a-zA-Z0-9_-]+)\}`)
)

// UnmarshalYAML sets default values for GroupConfig.
func (c *GroupConfig) UnmarshalYAML(unmarshal func(any) error) error {
	type rawGroupConfig GroupConfig
	defaults := rawGroupConfig{
		Swap:       true,
		Exclusive:  true,
		Persistent: false,
		Members:    []string{},
	}

	err := unmarshal(&defaults)
	if err != nil {
		return err
	}

	*c = GroupConfig(defaults)
	return nil
}

type HooksConfig struct {
	OnStartup HookOnStartup `yaml:"on_startup"`
}

type HookOnStartup struct {
	Preload []string `yaml:"preload"`
}

type Config struct {
	Models   map[string]ModelConfig `yaml:"models"` /* key is model ID */
	Profiles map[string][]string    `yaml:"profiles"`
	Groups   map[string]GroupConfig `yaml:"groups"` /* key is group ID */

	// map aliases to actual model IDs
	aliases map[string]string

	LogLevel      string `yaml:"logLevel"`
	LogTimeFormat string `yaml:"logTimeFormat"`

	// for key/value replacements in model's cmd, cmdStop, proxy, checkEndPoint
	Macros MacroList `yaml:"macros"`

	// hooks, see: #209
	Hooks HooksConfig `yaml:"hooks"`

	// automatic port assignments
	StartPort int `yaml:"startPort"`

	HealthCheckTimeout int  `yaml:"healthCheckTimeout"`
	MetricsMaxInMemory int  `yaml:"metricsMaxInMemory"`
	LogRequests        bool `yaml:"logRequests"`

	// send loading state in reasoning
	SendLoadingState bool `yaml:"sendLoadingState"`

	// present aliases to /v1/models OpenAI API listing
	IncludeAliasesInList bool `yaml:"includeAliasesInList"`
}

func (cfg *Config) RealModelName(search string) (string, bool) {
	if _, found := cfg.Models[search]; found {
		return search, true
	} else if name, found := cfg.aliases[search]; found {
		return name, found
	} else {
		return "", false
	}
}

func (cfg *Config) FindConfig(modelName string) (ModelConfig, string, bool) {
	if realName, found := cfg.RealModelName(modelName); !found {
		return ModelConfig{}, "", false
	} else {
		return cfg.Models[realName], realName, true
	}
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return LoadConfigFromReader(file)
}

func LoadConfigFromReader(r io.Reader) (*Config, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// default configuration values
	cfg := Config{
		HealthCheckTimeout: 120,
		StartPort:          5800,
		LogLevel:           "info",
		LogTimeFormat:      "",
		MetricsMaxInMemory: 1000,
	}
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	if cfg.HealthCheckTimeout < 15 {
		// set a minimum of 15 seconds
		cfg.HealthCheckTimeout = 15
	}

	if cfg.StartPort < 1 {
		return nil, errors.New("startPort must be greater than 1")
	}

	// Populate the aliases map
	cfg.aliases = make(map[string]string)
	for modelName, modelConfig := range cfg.Models {
		for _, alias := range modelConfig.Aliases {
			if _, found := cfg.aliases[alias]; found {
				return nil, fmt.Errorf("duplicate alias %s found in model: %s", alias, modelName)
			}
			cfg.aliases[alias] = modelName
		}
	}

	/* check macro constraint rules:

	- name must fit the regex ^[a-zA-Z0-9_-]+$
	- names must be less than 64 characters (no reason, just cause)
	- name can not be any reserved macros: PORT, MODEL_ID
	- macro values must be less than 1024 characters
	*/
	for _, macro := range cfg.Macros {
		err = validateMacro(macro.Name, macro.Value)
		if err != nil {
			return nil, err
		}
	}

	// Get and sort all model IDs first, makes testing more consistent
	modelIds := make([]string, 0, len(cfg.Models))
	for modelId := range cfg.Models {
		modelIds = append(modelIds, modelId)
	}
	sort.Strings(modelIds) // This guarantees stable iteration order

	nextPort := cfg.StartPort
	for _, modelId := range modelIds {
		modelConfig := cfg.Models[modelId]

		// Strip comments from command fields before macro expansion
		modelConfig.Cmd = StripComments(modelConfig.Cmd)
		modelConfig.CmdStop = StripComments(modelConfig.CmdStop)

		// validate model macros
		for _, macro := range modelConfig.Macros {
			err = validateMacro(macro.Name, macro.Value)
			if err != nil {
				return nil, fmt.Errorf("model %s: %s", modelId, err.Error())
			}
		}

		// Merge global config and model macros. Model macros take precedence
		mergedMacros := make(MacroList, 0, len(cfg.Macros)+len(modelConfig.Macros))
		mergedMacros = append(mergedMacros, MacroEntry{Name: "MODEL_ID", Value: modelId})

		// Add global macros first
		mergedMacros = append(mergedMacros, cfg.Macros...)

		// Add model macros (can override global)
		for _, entry := range modelConfig.Macros {
			// Remove any existing global macro with same name
			found := false
			for i, existing := range mergedMacros {
				if existing.Name == entry.Name {
					mergedMacros[i] = entry // Override
					found = true
					break
				}
			}
			if !found {
				mergedMacros = append(mergedMacros, entry)
			}
		}

		// First pass: Substitute user-defined macros in reverse order (LIFO - last defined first)
		// This allows later macros to reference earlier ones
		for i := len(mergedMacros) - 1; i >= 0; i-- {
			entry := mergedMacros[i]
			macroSlug := fmt.Sprintf("${%s}", entry.Name)
			macroStr := fmt.Sprintf("%v", entry.Value)

			// Substitute in command fields
			modelConfig.Cmd = strings.ReplaceAll(modelConfig.Cmd, macroSlug, macroStr)
			modelConfig.CmdStop = strings.ReplaceAll(modelConfig.CmdStop, macroSlug, macroStr)
			modelConfig.Proxy = strings.ReplaceAll(modelConfig.Proxy, macroSlug, macroStr)
			modelConfig.CheckEndpoint = strings.ReplaceAll(modelConfig.CheckEndpoint, macroSlug, macroStr)
			modelConfig.Filters.StripParams = strings.ReplaceAll(modelConfig.Filters.StripParams, macroSlug, macroStr)

			// Substitute in metadata (recursive)
			if len(modelConfig.Metadata) > 0 {
				var err error
				result, err := substituteMacroInValue(modelConfig.Metadata, entry.Name, entry.Value)
				if err != nil {
					return nil, fmt.Errorf("model %s metadata: %s", modelId, err.Error())
				}
				modelConfig.Metadata = result.(map[string]any)
			}
		}

		// Final pass: check if PORT macro is needed after macro expansion
		// ${PORT} is a resource on the local machine so a new port is only allocated
		// if it is required in either cmd or proxy keys
		cmdHasPort := strings.Contains(modelConfig.Cmd, "${PORT}")
		proxyHasPort := strings.Contains(modelConfig.Proxy, "${PORT}")
		if cmdHasPort || proxyHasPort { // either has it
			if !cmdHasPort && proxyHasPort { // but both don't have it
				return nil, fmt.Errorf("model %s: proxy uses ${PORT} but cmd does not - ${PORT} is only available when used in cmd", modelId)
			}

			// Add PORT macro and substitute it
			portEntry := MacroEntry{Name: "PORT", Value: nextPort}
			macroSlug := "${PORT}"
			macroStr := strconv.Itoa(nextPort)

			modelConfig.Cmd = strings.ReplaceAll(modelConfig.Cmd, macroSlug, macroStr)
			modelConfig.CmdStop = strings.ReplaceAll(modelConfig.CmdStop, macroSlug, macroStr)
			modelConfig.Proxy = strings.ReplaceAll(modelConfig.Proxy, macroSlug, macroStr)

			// Substitute PORT in metadata
			if len(modelConfig.Metadata) > 0 {
				var err error
				result, err := substituteMacroInValue(modelConfig.Metadata, portEntry.Name, portEntry.Value)
				if err != nil {
					return nil, fmt.Errorf("model %s metadata: %s", modelId, err.Error())
				}
				modelConfig.Metadata = result.(map[string]any)
			}

			nextPort++
		}

		// make sure there are no unknown macros that have not been replaced
		fieldMap := map[string]string{
			"cmd":                 modelConfig.Cmd,
			"cmdStop":             modelConfig.CmdStop,
			"proxy":               modelConfig.Proxy,
			"checkEndpoint":       modelConfig.CheckEndpoint,
			"filters.stripParams": modelConfig.Filters.StripParams,
		}

		for fieldName, fieldValue := range fieldMap {
			matches := macroPatternRegex.FindAllStringSubmatch(fieldValue, -1)
			for _, match := range matches {
				macroName := match[1]
				if macroName == "PID" && fieldName == "cmdStop" {
					continue // this is ok, has to be replaced by process later
				}
				// Reserved macros are always valid (they should have been substituted already)
				if macroName == "PORT" || macroName == "MODEL_ID" {
					return nil, fmt.Errorf("macro '${%s}' should have been substituted in %s.%s", macroName, modelId, fieldName)
				}
				// Any other macro is unknown
				return nil, fmt.Errorf("unknown macro '${%s}' found in %s.%s", macroName, modelId, fieldName)
			}
		}

		// Check for unknown macros in metadata
		if len(modelConfig.Metadata) > 0 {
			err := validateMetadataForUnknownMacros(modelConfig.Metadata, modelId)
			if err != nil {
				return nil, err
			}
		}

		// Validate the proxy URL.
		if _, err := url.Parse(modelConfig.Proxy); err != nil {
			return nil, fmt.Errorf(
				"model %s: invalid proxy URL: %w", modelId, err,
			)
		}

		// if sendLoadingState is nil, set it to the global config value
		// see #366
		if modelConfig.SendLoadingState == nil {
			v := cfg.SendLoadingState // copy it
			modelConfig.SendLoadingState = &v
		}

		cfg.Models[modelId] = modelConfig
	}

	cfg.AddDefaultGroup()

	// check that members are all unique in the groups
	memberUsage := make(map[string]string) // maps member to group it appears in
	for groupID, groupConfig := range cfg.Groups {
		prevSet := make(map[string]bool)
		for _, member := range groupConfig.Members {
			// Check for duplicates within this group
			if _, found := prevSet[member]; found {
				return nil, fmt.Errorf("duplicate model member %s found in group: %s", member, groupID)
			}
			prevSet[member] = true

			// Check if member is used in another group
			if existingGroup, exists := memberUsage[member]; exists {
				return nil, fmt.Errorf("model member %s is used in multiple groups: %s and %s", member, existingGroup, groupID)
			}
			memberUsage[member] = groupID
		}
	}

	// clean up hooks preload
	if len(cfg.Hooks.OnStartup.Preload) > 0 {
		var toPreload []string
		for _, modelID := range cfg.Hooks.OnStartup.Preload {
			modelID = strings.TrimSpace(modelID)
			if modelID == "" {
				continue
			}
			if realName, found := cfg.RealModelName(modelID); found {
				toPreload = append(toPreload, realName)
			}
		}

		cfg.Hooks.OnStartup.Preload = toPreload
	}

	return &cfg, nil
}

// AddDefaultGroup rewrites the yaml to include a default group with any orphaned models.
func (cfg *Config) AddDefaultGroup() {
	if cfg.Groups == nil {
		cfg.Groups = make(map[string]GroupConfig)
	}

	defaultGroup := GroupConfig{
		Swap:      true,
		Exclusive: true,
		Members:   []string{},
	}
	// if groups is empty, create a default group and put
	// all models into it
	if len(cfg.Groups) == 0 {
		for modelName := range cfg.Models {
			defaultGroup.Members = append(defaultGroup.Members, modelName)
		}
	} else {
		// iterate over existing group members and add non-grouped models into the default group
		for modelName := range cfg.Models {
			foundModel := false
		found:
			// search for the model in existing groups
			for _, groupConfig := range cfg.Groups {
				for _, member := range groupConfig.Members {
					if member == modelName {
						foundModel = true
						break found
					}
				}
			}

			if !foundModel {
				defaultGroup.Members = append(defaultGroup.Members, modelName)
			}
		}
	}

	sort.Strings(defaultGroup.Members) // make consistent ordering for testing
	cfg.Groups[DEFAULT_GROUP_ID] = defaultGroup
}

func SanitizeCommand(cmdStr string) ([]string, error) {
	var cleanedLines []string
	for line := range strings.SplitSeq(cmdStr, "\n") {
		trimmed := strings.TrimSpace(line)
		// Skip comment lines
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		// Handle trailing backslashes by replacing with space
		if before, ok := strings.CutSuffix(trimmed, "\\"); ok {
			cleanedLines = append(cleanedLines, before+" ")
		} else {
			cleanedLines = append(cleanedLines, line)
		}
	}

	// put it back together
	cmdStr = strings.Join(cleanedLines, "\n")

	// Split the command into arguments
	var args []string
	if runtime.GOOS == "windows" {
		args = shlex.Windows.Split(cmdStr)
	} else {
		args = shlex.Posix.Split(cmdStr)
	}

	// Ensure the command is not empty
	if len(args) == 0 {
		return nil, errors.New("empty command")
	}

	return args, nil
}

func StripComments(cmdStr string) string {
	var cleanedLines []string
	for line := range strings.SplitSeq(cmdStr, "\n") {
		trimmed := strings.TrimSpace(line)
		// Skip comment lines
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		cleanedLines = append(cleanedLines, line)
	}
	return strings.Join(cleanedLines, "\n")
}

// validateMacro validates macro name and value constraints.
func validateMacro(name string, value any) error {
	if len(name) >= 64 {
		return fmt.Errorf("macro name '%s' exceeds maximum length of 63 characters", name)
	}
	if !macroNameRegex.MatchString(name) {
		return fmt.Errorf("macro name '%s' contains invalid characters, must match pattern ^[a-zA-Z0-9_-]+$", name)
	}

	// Validate that value is a scalar type
	switch v := value.(type) {
	case string:
		if len(v) >= 1024 {
			return fmt.Errorf("macro value for '%s' exceeds maximum length of 1024 characters", name)
		}
		// Check for self-reference
		macroSlug := fmt.Sprintf("${%s}", name)
		if strings.Contains(v, macroSlug) {
			return fmt.Errorf("macro '%s' contains self-reference", name)
		}
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		// These types are allowed
	default:
		return fmt.Errorf("macro '%s' has invalid type %T, must be a scalar type (string, int, float, or bool)", name, value)
	}

	switch name {
	case "PORT", "MODEL_ID":
		return fmt.Errorf("macro name '%s' is reserved", name)
	}

	return nil
}

// validateMetadataForUnknownMacros recursively checks for any remaining macro references in metadata.
func validateMetadataForUnknownMacros(value any, modelId string) error {
	switch v := value.(type) {
	case string:
		matches := macroPatternRegex.FindAllStringSubmatch(v, -1)
		for _, match := range matches {
			macroName := match[1]
			return fmt.Errorf("model %s metadata: unknown macro '${%s}'", modelId, macroName)
		}
		return nil

	case map[string]any:
		for _, val := range v {
			err := validateMetadataForUnknownMacros(val, modelId)
			if err != nil {
				return err
			}
		}
		return nil

	case []any:
		for _, val := range v {
			err := validateMetadataForUnknownMacros(val, modelId)
			if err != nil {
				return err
			}
		}
		return nil

	default:
		// Scalar types don't contain macros
		return nil
	}
}

// substituteMacroInValue recursively substitutes a single macro in a value structure
// This is called once per macro, allowing LIFO substitution order.
func substituteMacroInValue(value any, macroName string, macroValue any) (any, error) {
	macroSlug := fmt.Sprintf("${%s}", macroName)
	macroStr := fmt.Sprintf("%v", macroValue)

	switch v := value.(type) {
	case string:
		// Check if this is a direct macro substitution
		if v == macroSlug {
			return macroValue, nil
		}
		// Handle string interpolation
		if strings.Contains(v, macroSlug) {
			return strings.ReplaceAll(v, macroSlug, macroStr), nil
		}
		return v, nil

	case map[string]any:
		// Recursively process map values
		newMap := make(map[string]any)
		for key, val := range v {
			newVal, err := substituteMacroInValue(val, macroName, macroValue)
			if err != nil {
				return nil, err
			}
			newMap[key] = newVal
		}
		return newMap, nil

	case []any:
		// Recursively process slice elements
		newSlice := make([]any, len(v))
		for i, val := range v {
			newVal, err := substituteMacroInValue(val, macroName, macroValue)
			if err != nil {
				return nil, err
			}
			newSlice[i] = newVal
		}
		return newSlice, nil

	default:
		// Return scalar types as-is
		return value, nil
	}
}

// MarshalYAML cannot guarantee the order because it returns map[string]any.
func (ml MacroList) MarshalYAML() (any, error) {
	return ml.ToMap(), nil
}
