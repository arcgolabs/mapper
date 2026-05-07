package mapper

// ValidationEngine validates destination values after mapping.
type ValidationEngine interface {
	Struct(any) error
}

// Config contains mapper behavior that affects field discovery and validation.
type Config struct {
	// TagName is the struct tag used to bind destination fields to source fields.
	// The default tag is "mapper".
	TagName string

	// PlanCacheSize is the maximum number of source/destination mapping plans
	// held in the mapper cache. The default is 1024.
	PlanCacheSize int

	// Strict reports an error when an exported destination field has no matching
	// source field. When false, unmatched destination fields are left unchanged.
	Strict bool
}

type settings struct {
	config      Config
	converters  []any
	beforeHooks []any
	afterHooks  []any
	validator   ValidationEngine
}

// Option configures a mapper call or Mapper instance.
type Option func(*settings)

func defaultConfig() Config {
	return Config{TagName: "mapper", PlanCacheSize: 1024}
}

func newSettings() settings {
	return settings{config: defaultConfig()}
}

// WithTagName changes the struct tag used for explicit field bindings.
func WithTagName(name string) Option {
	return func(s *settings) {
		if name != "" {
			s.config.TagName = name
		}
	}
}

// WithStrict toggles strict destination-field validation.
func WithStrict(strict bool) Option {
	return func(s *settings) {
		s.config.Strict = strict
	}
}

// WithPlanCacheSize changes the maximum number of cached mapping plans.
func WithPlanCacheSize(size int) Option {
	return func(s *settings) {
		if size > 0 {
			s.config.PlanCacheSize = size
		}
	}
}

// Strict reports an error when a destination field cannot be matched.
func Strict() Option {
	return WithStrict(true)
}

// Converter registers a converter for a single mapper call or Mapper instance.
func Converter[S, D any](fn func(S) D) Option {
	return func(s *settings) {
		s.converters = append(s.converters, fn)
	}
}

// ConverterE registers a converter that can return an error.
func ConverterE[S, D any](fn func(S) (D, error)) Option {
	return func(s *settings) {
		s.converters = append(s.converters, fn)
	}
}

// ConverterFunc registers a converter using reflection. It must be a function
// with one input and either one output or an output plus error.
func ConverterFunc(fn any) Option {
	return func(s *settings) {
		s.converters = append(s.converters, fn)
	}
}

// WithValidator enables destination validation using a custom validator implementation.
// The type must provide a Struct(any) error method.
// Set to nil to keep the default (no validation).
func WithValidator(v ValidationEngine) Option {
	return func(s *settings) {
		s.validator = v
	}
}

// BeforeMap registers a hook that runs before a top-level mapping operation.
func BeforeMap[S, D any](fn func(S, *D)) Option {
	return func(s *settings) {
		s.beforeHooks = append(s.beforeHooks, fn)
	}
}

// BeforeMapE registers an error-returning hook that runs before top-level mapping.
func BeforeMapE[S, D any](fn func(S, *D) error) Option {
	return func(s *settings) {
		s.beforeHooks = append(s.beforeHooks, fn)
	}
}

// AfterMap registers a hook that runs after a successful top-level mapping operation.
func AfterMap[S, D any](fn func(S, *D)) Option {
	return func(s *settings) {
		s.afterHooks = append(s.afterHooks, fn)
	}
}

// AfterMapE registers an error-returning hook that runs after successful mapping.
func AfterMapE[S, D any](fn func(S, *D) error) Option {
	return func(s *settings) {
		s.afterHooks = append(s.afterHooks, fn)
	}
}

// BeforeMapFunc registers a before hook using reflection. It must be a function
// with source and destination-pointer inputs, and either no return value or error.
func BeforeMapFunc(fn any) Option {
	return func(s *settings) {
		s.beforeHooks = append(s.beforeHooks, fn)
	}
}

// AfterMapFunc registers an after hook using reflection. It must be a function
// with source and destination-pointer inputs, and either no return value or error.
func AfterMapFunc(fn any) Option {
	return func(s *settings) {
		s.afterHooks = append(s.afterHooks, fn)
	}
}
