package provider

type Params struct {
	Context  *Context
	Model    string
	Messages []Message
}

type paramsOptions func(*Params)

func NewParams(opts ...paramsOptions) *Params {
	params := &Params{}

	for _, opt := range opts {
		if opt != nil {
			opt(params)
		}
	}

	return params
}

func ParamsWithContext(ctx *Context) paramsOptions {
	return func(params *Params) {
		params.Context = ctx
	}
}

func ParamsWithModel(model string) paramsOptions {
	return func(params *Params) {
		params.Model = model
	}
}
