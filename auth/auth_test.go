package auth_test

import (
	"fmt"
	"github.com/go-chassis/foundation/httpclient"
	"github.com/go-chassis/go-chassis-cloud/auth"
	"github.com/go-chassis/go-chassis/v2/core/common"
	"github.com/go-chassis/go-chassis/v2/core/config"
	"github.com/go-chassis/go-chassis/v2/core/config/model"
	_ "github.com/go-chassis/go-chassis/v2/security/cipher/plugins/aes"
	_ "github.com/go-chassis/go-chassis/v2/security/cipher/plugins/plain"
	"github.com/stretchr/testify/assert"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func testWriteFile(t *testing.T, name string, ak, sk, project, cipher string) {
	contentFormat := `---
cse:
  credentials:
    accessKey: %s
    secretKey: %s
    project: %s
    akskCustomCipher: %s
`
	content := fmt.Sprintf(contentFormat, ak, sk, project, cipher)
	f, err := os.OpenFile(name, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0700)
	assert.NoError(t, err)
	defer f.Close()
	_, err = f.WriteString(content)
	assert.NoError(t, err)
}

func testCheckAkAndProject(t *testing.T, ak, project string) {
	req, err := http.NewRequest("GET", "http://127.0.0.1:8080", nil)
	assert.NoError(t, err)
	assert.NoError(t, err)
	httpclient.SignRequest(req)
	assert.Equal(t, req.Header.Get(auth.HeaderServiceAk), ak)
	assert.Equal(t, req.Header.Get(auth.HeaderServiceProject), project)
}

func testAuthNotLoaded(t *testing.T) {
	r, err := http.NewRequest("GET", "http://127.0.0.1:8080", nil)
	assert.NoError(t, err)
	httpclient.SignRequest(r)
	assert.Equal(t, 0, len(r.Header))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(r.Header))
}

func Test_loadAkskAuth(t *testing.T) {
	utDir := filepath.Join(os.Getenv("GOPATH"), "test")
	authTestDir := filepath.Join(utDir, "auth")
	chassisHome := authTestDir
	cipherRootDir := filepath.Join(authTestDir, "cipher")
	os.Setenv("CHASSIS_HOME", chassisHome)
	chassisConf := filepath.Join(chassisHome, "conf")
	err := os.MkdirAll(chassisConf, 0700)
	assert.NoError(t, err)
	os.Setenv(auth.CipherRootEnv, cipherRootDir)
	err = os.MkdirAll(cipherRootDir, 0700)
	assert.NoError(t, err)

	chassisFilePath := filepath.Join(chassisConf, "chassis.yaml")
	microserviceFilePath := filepath.Join(chassisConf, "microservice.yaml")
	os.Create(chassisFilePath)
	os.Create(microserviceFilePath)
	credentialFilePath := filepath.Join(cipherRootDir, auth.KeytoolAkskFile)
	uriWithProjectCnNorth := "https://cse.cn-north-1.myhwclouds.com:443"

	t.Log("Get aksk config from chassis.yaml")
	ak, sk, project, cipherName := "a0", "s0", "p0", ""
	testWriteFile(t, chassisFilePath, ak, sk, project, cipherName)

	// rm certificate.yaml
	_, err = os.Stat(credentialFilePath)
	if err != nil {
		assert.True(t, os.IsNotExist(err))
		if !os.IsNotExist(err) {
			t.Fail()
		}
	} else {
		e := os.Remove(credentialFilePath)
		assert.NoError(t, e)
		if e != nil {
			t.Fail()
		}
	}

	config.InitArchaius()
	config.GlobalDefinition = &model.GlobalCfg{}
	config.GlobalDefinition.ServiceComb.Registry.Address = uriWithProjectCnNorth
	err = auth.LoadAkskAuth()
	assert.NoError(t, err)
	testCheckAkAndProject(t, ak, project)

	t.Log("Get aksk config from CIPHER_ROOT/certificate.yaml")
	ak, sk, project, cipherName = "a1", "s1", "p1", ""
	testWriteFile(t, credentialFilePath, ak, sk, project, cipherName)
	err = auth.LoadAkskAuth()
	assert.NoError(t, err)
	testCheckAkAndProject(t, ak, project)

	t.Log("One of ak and sk is empty")
	httpclient.SignRequest = func(*http.Request) error { return nil }
	ak, sk, project, cipherName = "a2", "", "p2", ""
	testWriteFile(t, credentialFilePath, ak, sk, project, cipherName)
	err = auth.LoadAkskAuth()
	assert.Error(t, err)
	assert.NotEqual(t, auth.ErrAuthConfNotExist, err)
	testAuthNotLoaded(t)

	t.Log("Ak sk not exists")
	httpclient.SignRequest = func(*http.Request) error { return nil }
	ak, sk, project, cipherName = "", "", "p3", ""
	testWriteFile(t, credentialFilePath, ak, sk, project, cipherName)
	err = auth.LoadAkskAuth()
	assert.Error(t, err)
	assert.Equal(t, auth.ErrAuthConfNotExist, err)
	testAuthNotLoaded(t)

	t.Log("AkskCustomCipher exists")
	ak, sk, project, cipherName = "a4", "s4", "p4", "default"
	testWriteFile(t, credentialFilePath, ak, sk, project, cipherName)
	err = auth.LoadAkskAuth()
	assert.NoError(t, err)
	testCheckAkAndProject(t, ak, project)

	t.Log("AkskCustomCipher not exists")
	httpclient.SignRequest = func(*http.Request) error { return nil }
	ak, sk, project, cipherName = "a5", "s5", "p5", "c5"
	testWriteFile(t, credentialFilePath, ak, sk, project, cipherName)
	err = auth.LoadAkskAuth()
	assert.Error(t, err)
	assert.NotEqual(t, auth.ErrAuthConfNotExist, err)
	testAuthNotLoaded(t)

	t.Log("Get project from uri")
	ak, sk, project, cipherName = "a6", "s6", "", ""
	testWriteFile(t, credentialFilePath, ak, sk, project, cipherName)
	err = auth.LoadAkskAuth()
	assert.NoError(t, err)
	testCheckAkAndProject(t, ak, "cn-north-1")

	t.Log("ServiceComb uri invalid")
	httpclient.SignRequest = func(*http.Request) error { return nil }
	ak, sk, project, cipherName = "a7", "s7", "", ""
	testWriteFile(t, credentialFilePath, ak, sk, project, cipherName)
	config.GlobalDefinition.ServiceComb.Registry.Address = ":://a+b"
	err = auth.LoadAkskAuth()
	assert.Error(t, err)
	assert.NotEqual(t, auth.ErrAuthConfNotExist, err)
	testAuthNotLoaded(t)

	t.Log("Get project from config")
	ak, sk, project, cipherName = "a9", "s9", "p9", ""
	testWriteFile(t, credentialFilePath, ak, sk, project, cipherName)
	err = auth.LoadAkskAuth()
	assert.NoError(t, err)
	testCheckAkAndProject(t, ak, project)

	t.Log("Use default project")
	config.GlobalDefinition.ServiceComb.Registry.Address = "http://cse:8080"
	ak, sk, project, cipherName = "a10", "s10", "", ""
	testWriteFile(t, credentialFilePath, ak, sk, project, cipherName)
	err = auth.LoadAkskAuth()
	assert.NoError(t, err)
	testCheckAkAndProject(t, ak, common.DefaultValue)
}
