package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/knadh/koanf/maps"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type parseOption struct {
	envPrefix      string
	delimiter      string
	tag            string
	flatStruct     bool
	helpFlag       string
	descriptionTag string
	flagTag        string
	shortTag       string
}

type ParseOption func(*parseOption)

func WithEnvPrefix(prefix string) ParseOption {
	return func(po *parseOption) {
		po.envPrefix = prefix
	}
}

func WithDelimiter(delimiter string) ParseOption {
	return func(po *parseOption) {
		po.delimiter = delimiter
	}
}

func WithStructTagName(tag string) ParseOption {
	return func(po *parseOption) {
		po.tag = tag
	}
}

func WithDescriptionStructTagName(tag string) ParseOption {
	return func(po *parseOption) {
		po.descriptionTag = tag
	}
}

type Validatable interface {
	Validate() error
}

// Parse takes every object and is able to fill and validate that object depending on config file, env file and flag values.
// https://github.com/knadh/koanf
// Your passed struct must define . delimited koanf struct tags in order to match env/.env and flag values to your struct.
// Additionally your struct may define a Validate() error method which is called at the end of parsing the config
// Registers flags and returns a parser function that can be used as PreRunE.
func RegisterFlags[T any](config *T, persistent bool, app *cobra.Command, options ...ParseOption) func() error {

	op := parseOption{
		envPrefix:      "DIFF_",
		delimiter:      ".",
		tag:            "koanf",
		flatStruct:     true,
		helpFlag:       "help",
		descriptionTag: "description",
		flagTag:        "flag",
		shortTag:       "short",
	}

	for _, o := range options {
		o(&op)
	}

	envToKoanf := func(s string) string {
		return strings.ToLower(strings.ReplaceAll(s, "_", op.delimiter))
	}

	f := func(s string) string {
		return envToKoanf(strings.TrimPrefix(s, op.envPrefix))
	}

	koanfToEnv := func(s string) string {
		return op.envPrefix + strings.ToUpper(strings.ReplaceAll(s, op.delimiter, "_"))
	}

	defaults := koanf.New(op.delimiter)

	// does not error
	_ = defaults.Load(structs.ProviderWithDelim(config, op.tag, op.delimiter), nil)

	var fs *pflag.FlagSet
	if persistent {
		fs = app.PersistentFlags()
	} else {
		fs = app.Flags()
	}

	ct := reflect.TypeOf(config)
	if ct.Kind() == reflect.Pointer {
		ct = ct.Elem()
	}

	defaultMap, _ := maps.Flatten(defaults.All(), nil, op.delimiter)
	maxKeyLen := maxKeyLen(defaultMap)
	padding := maxKeyLen + len(op.envPrefix) + 1
	format := "\n  %-" + strconv.Itoa(padding) + "s   %s"
	var sb strings.Builder
	sb.Grow((padding + 6) * len(defaultMap) * 3)

	// register flags for all known struct fields
	for i := 0; i < ct.NumField(); i++ {
		field := ct.Field(i)
		sTag := field.Tag
		key, found := sTag.Lookup(op.tag)
		if !found {
			continue
		}
		v := defaultMap[key]

		desc := sTag.Get(op.descriptionTag)
		short := sTag.Get(op.shortTag)
		flag := sTag.Get(op.flagTag)

		envName := koanfToEnv(key)
		// key, description
		sb.WriteString(fmt.Sprintf(format, envName, desc))

		// allow skipping of flag creation for specific fields
		if flag == "false" {
			continue
		}

		// key is now a flag name
		flagName := strings.ReplaceAll(key, op.delimiter, "-")

		if v != nil {
			// default value if not empty
			defaultVal := fmt.Sprintf("%v", v)
			if defaultVal != "" {
				sb.WriteString(fmt.Sprintf(" (default: %q)", defaultVal))
			}
		}

		switch x := v.(type) {
		case bool:
			if len(short) == 1 {
				fs.BoolP(flagName, short, x, desc)
			} else {
				fs.Bool(flagName, x, desc)
			}
		default:

			strValue := ""
			if v != nil {
				strValue = fmt.Sprintf("%v", v)
			}

			if len(short) == 1 {
				fs.StringP(flagName, short, strValue, desc)
			} else {
				fs.String(flagName, strValue, desc)
			}
		}
	}

	sb.WriteString("\n")
	app.Long += sb.String()

	return func() error {

		environment := koanf.New(op.delimiter)
		err := environment.Load(env.Provider(op.envPrefix, op.delimiter, f), nil)
		if err != nil {
			return err
		}

		// disable unknonwn flags errors
		before := fs.ParseErrorsWhitelist
		fs.ParseErrorsWhitelist.UnknownFlags = true
		defer func() {
			fs.ParseErrorsWhitelist = before
		}()

		err = fs.Parse(os.Args)
		if err != nil {
			return fmt.Errorf("failed to parse config flags: %w", err)
		}

		flagSet := koanf.New(op.delimiter)
		err = flagSet.Load(
			posflag.ProviderWithValue(
				fs,
				op.delimiter,
				nil,
				func(key, value string) (string, interface{}) {
					return envToKoanf(key), value
				},
			), nil)
		if err != nil {
			return err
		}

		// skip parsing of the config in case we encounter the help flag
		if flagSet.Bool(op.helpFlag) {
			return nil
		}

		dotenvFile := koanf.New(op.delimiter)

		k := koanf.New(op.delimiter)
		err = k.Merge(defaults)
		if err != nil {
			return err
		}

		// only support .env files format
		err = k.Merge(dotenvFile)
		if err != nil {
			return err
		}

		err = k.Merge(environment)
		if err != nil {
			return err
		}

		// merge flag map into struct map
		_ = k.Load(confmap.Provider(flagSet.All(), "-"), nil)

		err = k.UnmarshalWithConf("", config, koanf.UnmarshalConf{
			FlatPaths: op.flatStruct,
		})
		if err != nil {
			return err
		}

		var a any = config
		if v, ok := a.(Validatable); ok {
			return v.Validate()
		}
		return nil
	}
}

func maxKeyLen(m map[string]any) int {
	maxLen := 1
	for k := range m {
		keyLen := len(k)
		if keyLen > maxLen {
			maxLen = keyLen
		}
	}

	return maxLen
}
