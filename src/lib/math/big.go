package lib_math

import (
	"database/sql/driver"
	"fmt"
	"math/big"
)

type BigInt struct {
	big.Int
}

func NewBigInt(i *big.Int) *BigInt {
	v := &BigInt{}
	if i != nil {
		v.Set(i)
	}
	return v
}
func (self *BigInt) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, self.String())), nil
}
func (self *BigInt) UnmarshalJSON(text []byte) error {
	s := string(text)
	if s[0] == '"' {
		s = s[1 : len(s)-1]
	}
	self.SetString(s, 10)
	return nil
}
func (self BigInt) Value() (driver.Value, error) {
	return driver.Value(self.String()), nil
}
func (self *BigInt) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch value.(type) {
	case []uint8:
		s := string(value.([]uint8))
		self.SetString(s, 10)
	case string:
		self.SetString(value.(string), 10)
	case *big.Int:
		self.Set(value.(*big.Int))
	case BigInt:
		v := value.(BigInt)
		self.Set(&v.Int)
	case *BigInt:
		v := value.(*BigInt)
		self.Set(&v.Int)
	}
	return nil
}
