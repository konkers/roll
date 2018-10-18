package roll

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/asdine/storm"
	"github.com/ybbus/jsonrpc"
)

func getTestHttpClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport)
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	return &http.Client{
		Transport: transport,
	}

}

func getTestRpcClient(bot *Bot) jsonrpc.RPCClient {
	httpClient := getTestHttpClient()
	return jsonrpc.NewClientWithOpts("https://"+bot.Config.HTTPSAddr+"/rpc",
		&jsonrpc.RPCClientOpts{HTTPClient: httpClient})
}

func TestWebStartupTLSError(t *testing.T) {
	b, _ := newTestBot(t)
	b.Config.CertFile = ""
	err := b.Connect()
	if err == nil {
		t.Errorf("Did not get expected error from missing cert file.")
	}

	b, _ = newTestBot(t)
	b.Config.HTTPSAddr = "asdjflksdjfsd:1023210"
	err = b.Connect()
	if err == nil {
		t.Errorf("Did not get expected error from missing cert file.")
	}
}

type TestRpcService struct{}
type TestRpcModule struct {
	service *TestRpcService
}

func NewTestRpcModule(bot *Bot, dbBucket storm.Node) (Module, error) {
	return &TestRpcModule{
		service: &TestRpcService{},
	}, nil
}

func (m *TestRpcModule) Start() error {
	return nil
}

func (m *TestRpcModule) Stop() error {
	return nil
}

func (m *TestRpcModule) GetRPCService() interface{} {
	return m.service
}

func (s *TestRpcService) Inc(r *http.Request, in *int, out *int) error {
	*out = *in + 1
	return nil
}

func TestWebRpc(t *testing.T) {
	err := RegisterModuleFactory(NewTestRpcModule, "test_rpc")
	if err != nil {
		t.Fatalf("Unexpected error from RegisterModuleFactory(): %v", err)
	}

	bot, _ := newTestBot(t)
	err = bot.AddModule("test_rpc")
	if err != nil {
		t.Errorf("Unexpected error from AddModule(): %v", err)
	}

	err = bot.Connect()
	if err != nil {
		t.Errorf("Unexpected error from bot.Connect(): %v", err)
	}

	rpcClient := getTestRpcClient(bot)
	var out int
	resp, err := rpcClient.Call("test_rpc.Inc", int(1))
	if err != nil {
		t.Fatalf("Unexpected error from Rpc.Call(): %v", err)
	}
	err = resp.GetObject(&out)
	if err != nil {
		t.Errorf("Unexpected error from GetObject(): %v", err)
	}

	if out != 2 {
		t.Errorf("test_rpc.Inc(1) returned %d instead of 2", out)
	}

}

func TestWebRedirect(t *testing.T) {
	bot, _ := newConnectedTestBot(t)
	httpClient := getTestHttpClient()
	httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	path := "/test?foo"
	httpURL := "http://" + bot.Config.HTTPAddr + path
	httpsURL := "https://" + bot.Config.HTTPSAddr + path
	resp, err := httpClient.Get(httpURL)
	if err != nil {
		t.Fatalf("error getting %s: %v", httpURL, err)
	}

	if resp.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("Did not get status %d.  Got %s instead.", http.StatusTemporaryRedirect,
			resp.Status)
	}

	location := resp.Header.Get("Location")
	if location != httpsURL {
		t.Fatalf("Redirect location is %s instead of expected %s.", location, httpsURL)
	}
}

func testTimeError(t *testing.T, enc string) {
	var time2 Time
	var err error
	err = json.Unmarshal([]byte(enc), &time2)
	if err == nil {
		t.Errorf("Expected error decoding time %s: %v", enc, err)
	}
}

func testTimeSuccess(t *testing.T, enc string) {
	var time2 Time
	var err error
	enc = "\"" + enc + "\""
	err = json.Unmarshal([]byte(enc), &time2)
	if err != nil {
		t.Errorf("Error decoding %v: %v", enc, err)
		return
	}

	var d []byte
	d, err = json.Marshal(&time2)
	if err != nil {
		t.Errorf("Error encoding %v: %v", time2.Time, err)
		return
	}
	if string(d) != enc {
		t.Errorf("Encoded time is %s not expected %s", string(d), enc)
	}
}

func testDurationError(t *testing.T, enc string) {
	var dur Duration
	var err error
	err = json.Unmarshal([]byte(enc), &dur)
	if err == nil {
		t.Errorf("Expected error decoding duration %s: %v", enc, err)
	}
}

func testDurationSuccess(t *testing.T, enc string, expected string) {
	var dur Duration
	var err error
	enc = "\"" + enc + "\""
	expected = "\"" + expected + "\""
	err = json.Unmarshal([]byte(enc), &dur)
	if err != nil {
		t.Errorf("Error decoding %v: %v", enc, err)
		return
	}

	var d []byte
	d, err = json.Marshal(&dur)
	if err != nil {
		t.Errorf("Error encoding %v: %v", dur.Duration, err)
		return
	}
	if string(d) != expected {
		t.Errorf("Encoded time is %s not expected %s", string(d), expected)
	}
}

func TestTimeDurationJSON(t *testing.T) {
	testTimeSuccess(t, "Mon, 02 Jan 2006 15:04:05 MST")
	testTimeSuccess(t, "Wed, 04 Jan 2006 04:15:59 PST")
	testTimeError(t, "\"\"")
	testTimeError(t, "1")

	testDurationSuccess(t, "10s", "10s")
	testDurationSuccess(t, "15m", "15m0s")
	testDurationSuccess(t, "15m0s", "15m0s")
	testDurationSuccess(t, "15m10s", "15m10s")
	testDurationSuccess(t, "2h", "2h0m0s")
	testDurationSuccess(t, "2h0m0s", "2h0m0s")

	testDurationError(t, "\"5\"")
	testDurationError(t, "\"\"")
	testDurationError(t, "1")
}

func TestIndex(t *testing.T) {
	bot, _ := newConnectedTestBot(t)
	client := getTestHttpClient()
	url := "https://" + bot.Config.HTTPSAddr + "/"
	_, err := client.Get(url)
	if err != nil {
		t.Errorf("Got error getting %s: %v", url, err)
	}
}
