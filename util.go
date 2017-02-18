package main

import "strconv"

type jsonBool string

func (b jsonBool) Bool() (bool, error) {
	if b == "" {
		return false, nil
	}

	return strconv.ParseBool(string(b))
}
