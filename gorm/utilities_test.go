package gorm

import (
	"context"
	"reflect"
	"testing"

	"time"

	"github.com/stretchr/testify/assert"
)

type Human struct {
	Name  string
	Age   uint32 `gorm:"column:years"`
	Child *Child
}

func (p Human) TableName() string {
	return "db_humans"
}

type Child struct {
	Name string
}

func TestHandleFieldPath(t *testing.T) {
	tests := []struct {
		fieldPath []string
		dbName    string
		assoc     string
		err       bool
	}{
		{[]string{"name"}, "db_humans.name", "", false},
		{[]string{"age"}, "db_humans.years", "", false},
		{[]string{"child", "name"}, "child.name", "Child", false},
		{[]string{}, "", "", true},
	}
	for _, test := range tests {
		dbName, assoc, err := HandleFieldPath(context.Background(), test.fieldPath, &Human{})
		if test.err {
			assert.Equal(t, "", dbName)
			assert.Equal(t, "", assoc)
			assert.NotNil(t, err)
		} else {
			assert.Equal(t, test.dbName, dbName)
			assert.Equal(t, test.assoc, assoc)
			assert.Nil(t, err)
		}
	}
}

func TestIsModel(t *testing.T) {
	for tName, tCase := range map[string]struct {
		input reflect.Type
		want  bool
	}{
		"time.Time":  {reflect.TypeOf(time.Time{}), false},
		"*time.Time": {reflect.TypeOf(&time.Time{}), false},
	} {
		t.Run(tName, func(t *testing.T) {
			if got, want := isModel(tCase.input), tCase.want; got != want {
				t.Errorf("got %v; want %v", got, want)
			}
		})
	}
}
