package pgservicefile_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Meysam-sadeghi/pgservicefile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseServicefile(t *testing.T) {
	buf := bytes.NewBufferString(`# A comment
[abc]
host=abc.example.com
port=9999
dbname=abcdb
user=abcuser
# Another comment

[def]
host = def.example.com
dbname = defdb
user = defuser
application_name = has space
`)

	servicefile, err := pgservicefile.ParseServicefile(buf)
	require.NoError(t, err)
	require.NotNil(t, servicefile)

	assert.Len(t, servicefile.Services, 2)
	assert.Equal(t, "abc", servicefile.Services[0].Name)
	assert.Equal(t, "def", servicefile.Services[1].Name)

	abc, err := servicefile.GetService("abc")
	require.NoError(t, err)
	assert.Equal(t, servicefile.Services[0], abc)
	assert.Len(t, abc.Settings, 4)
	assert.Equal(t, "abc.example.com", abc.Settings["host"])
	assert.Equal(t, "9999", abc.Settings["port"])
	assert.Equal(t, "abcdb", abc.Settings["dbname"])
	assert.Equal(t, "abcuser", abc.Settings["user"])

	def, err := servicefile.GetService("def")
	require.NoError(t, err)
	assert.Equal(t, servicefile.Services[1], def)
	assert.Len(t, def.Settings, 4)
	assert.Equal(t, "def.example.com", def.Settings["host"])
	assert.Equal(t, "defdb", def.Settings["dbname"])
	assert.Equal(t, "defuser", def.Settings["user"])
	assert.Equal(t, "has space", def.Settings["application_name"])
}

func TestParseServicefileWithInvalidFile(t *testing.T) {
	buf := bytes.NewBufferString("Invalid syntax\n")

	servicefile, err := pgservicefile.ParseServicefile(buf)
	assert.Error(t, err)
	assert.Nil(t, servicefile)
}

// https://github.com/jackc/pgservicefile/issues/5
func TestParseServicefileWithMissedServiceSection(t *testing.T) {
	buf := bytes.NewBufferString("a = b\n")

	servicefile, err := pgservicefile.ParseServicefile(buf)
	assert.Error(t, err)
	assert.Nil(t, servicefile)
}

func TestSecurityHardening(t *testing.T) {
	var nilSF *pgservicefile.Servicefile
	_, err := nilSF.GetService("abc")
	require.Error(t, err)

	_, err = pgservicefile.ParseServicefile(nil)
	require.Error(t, err)

	for _, invalid := range []string{
		"[]\nhost=h\n",
		"[svc]\n=val\n",
		"[svc]\nhost=a\x00b\n",
	} {
		_, err := pgservicefile.ParseServicefile(strings.NewReader(invalid))
		require.Error(t, err)
	}

	oversized := strings.NewReader("[svc]\nhost=" + strings.Repeat("a", (4<<20)+16))
	_, err = pgservicefile.ParseServicefile(oversized)
	require.Error(t, err)

	dir := t.TempDir()
	_, err = pgservicefile.ReadServicefile(dir)
	require.Error(t, err)

	validPath := filepath.Join(dir, "pg_service.conf")
	require.NoError(t, os.WriteFile(validPath, []byte("[prod]\nhost=127.0.0.1\nport=5432\n"), 0o600))
	sf, err := pgservicefile.ReadServicefile(validPath)
	require.NoError(t, err)
	svc, err := sf.GetService("prod")
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1", svc.Settings["host"])
}
