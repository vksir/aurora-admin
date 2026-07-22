package steam

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetPatchNotesRSS(t *testing.T) {
	feed, err := GetPatchNotesRSS("343050")
	assert.Nil(t, err)
	fmt.Println(feed)
}
