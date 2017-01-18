package argparse

type Args []Parser

var _ Parser = Args{}

func (args Args) Parse(str string) (res interface{}, nparsed int, err error) {
	values := make([]interface{}, len(args))
	inLen := len(str)
	for i, arg := range args {
		nparsed += countLeftSpaces(str[nparsed:])
		if nparsed >= inLen {
			err = ErrTooFewArgs
			return
		}
		val, n, e := arg.Parse(str[nparsed:])
		if e != nil {
			err = e
			return
		}
		nparsed += n
		values[i] = val
	}
	nparsed += countLeftSpaces(str[nparsed:])
	res = values
	if nparsed != inLen {
		err = ErrTooManyArgs
	}
	return
}
