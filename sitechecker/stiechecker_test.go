package sitechecker

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCheckIsUrlHttp(t *testing.T) {
	assert := assert.New(t)
	url := "http://www.google.com"
	result := CheckIsUrl(url)
	assert.True(result)
}

func TestCheckIsUrlHttps(t *testing.T) {
	assert := assert.New(t)
	url := "https://www.google.com"
	result := CheckIsUrl(url)
	assert.True(result)
}

func TestCheckIsUrlNotHttp(t *testing.T) {
	assert := assert.New(t)
	url := "asdfasdfas"
	result := CheckIsUrl(url)
	assert.False(result)
}
