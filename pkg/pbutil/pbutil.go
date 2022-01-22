package pbutil

import (
	"fmt"
)

type Marshaler interface {
	Marshal() (data []byte, err error)
}

type Unmarshaler interface {
	Unmarshal(data []byte) error
}

func MustUnmarshal(um Unmarshaler, data []byte) {
	if err := um.Unmarshal(data); err != nil {
		panic(fmt.Sprintf("unmarshal should never fail (%v)", err))
	}
}
