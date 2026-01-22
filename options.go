package pongo2

// Options allow you to change the behavior of template-engine. You can change
// the options before calling the Execute method.
type Options struct {
	// If this is set to true the first newline after a block is removed (block,
	// not variable tag!). Defaults to false.
	TrimBlocks bool

	// If this is set to true leading spaces and tabs are stripped from the
	// start of a line to a block. Defaults to false
	LStripBlocks bool

	// Sets the name of the filter used to escape values. Defaults to "escape",
	// which escapes HTML sequences.
	AutoescapeFilter string

	// If this is set to true, variables that resolve to *Template or string values containing template tags are
	// further resolved.
	DeepResolve bool

	// If this is set to true, functions directly assigned as context variables cannot be called.
	DisableContextFunctions bool

	// If this is set to true, functions within context variables (such as struct member functions) cannot be called.
	DisableNestedFunctions bool

	// If this is set to true, struct fields, map keys, and variable names will be treated as case-insensitive.
	IgnoreVariableCase bool

	// Assigns a translation function to be used for the translate tag.
	Translator TranslateFunc
}

func newOptions() *Options {
	return &Options{
		TrimBlocks:              false,
		LStripBlocks:            false,
		AutoescapeFilter:        "escape",
		DeepResolve:             false,
		DisableContextFunctions: false,
		DisableNestedFunctions:  false,
		IgnoreVariableCase:      false,
	}
}

// Update updates this options from another options.
func (opt *Options) Update(other *Options) *Options {
	opt.TrimBlocks = other.TrimBlocks
	opt.LStripBlocks = other.LStripBlocks
	opt.AutoescapeFilter = other.AutoescapeFilter
	opt.DeepResolve = other.DeepResolve
	opt.DisableContextFunctions = other.DisableContextFunctions
	opt.DisableNestedFunctions = other.DisableNestedFunctions
	opt.IgnoreVariableCase = other.IgnoreVariableCase
	opt.Translator = other.Translator

	return opt
}
