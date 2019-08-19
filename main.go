package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
)

var c = map[string]string{
	"AZURE_CLIENT_ID":         os.Getenv("AZURE_CLIENT_ID"),
	"AZURE_CLIENT_SECRET":     os.Getenv("AZURE_CLIENT_SECRET"),
	"AZURE_SUBSCRIPTION_ID":   os.Getenv("AZURE_SUBSCRIPTION_ID"),
	"AZURE_TENANT_ID":         os.Getenv("AZURE_TENANT_ID"),
	"RESOURCE":                os.Getenv("RESOURCE"),
	"ACTIVEDIRECTORYENDPOINT": os.Getenv("ACTIVEDIRECTORYENDPOINT"),
	"AZUREDATFACTORYHOST":     os.Getenv("AZUREDATFACTORYHOST"),
}

func main() {

	http.HandleFunc("/", proxy)
	http.HandleFunc("/health/pulse/", pulse)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func proxy(w http.ResponseWriter, r *http.Request) {
	token := returnToken()
	authorization := strings.Join([]string{"Bearer", " ", token}, "")

	form, _ := url.ParseQuery(r.URL.RawQuery)
	form.Add("api-version", r.FormValue("api-version"))
	r.URL.RawQuery = form.Encode()

	requestURL := strings.Join([]string{c["AZUREDATFACTORYHOST"], r.RequestURI}, "")

	req, err := http.NewRequest(r.Method, requestURL, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", authorization)

	if r.Method == "POST" {
		bodyBuffer, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatal(err)
		} else {
			log.Print(strings.Join([]string{"Body:", string(bodyBuffer)}, " "))
		}

		req.Body = ioutil.NopCloser(strings.NewReader(string(bodyBuffer)))
	}

	bytes, err := doRequest(req)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Print(strings.Join([]string{"Method:", r.Method, "RequestURI:", r.RequestURI}, " "))
	}

	fmt.Fprintf(w, string(bytes))
}

func pulse(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "{\"status\": \"OK\"}")
}

func doRequest(req *http.Request) ([]byte, error) {
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	if 200 != resp.StatusCode {
		return nil, fmt.Errorf("%s", body)
	}
	return body, nil
}

func newServicePrincipalTokenFromCredentials(c map[string]string, scope string) (*adal.ServicePrincipalToken, error) {
	oauthConfig, err := adal.NewOAuthConfig(c["ACTIVEDIRECTORYENDPOINT"], c["AZURE_TENANT_ID"])
	if err != nil {
		log.Fatal(err)
	}

	return adal.NewServicePrincipalToken(*oauthConfig, c["AZURE_CLIENT_ID"], c["AZURE_CLIENT_SECRET"], c["RESOURCE"])
}

func returnToken() string {
	spt, err := newServicePrincipalTokenFromCredentials(c, azure.PublicCloud.ResourceManagerEndpoint)

	if err != nil {
		log.Fatal(err)
	}

	sptError := spt.Refresh()
	if sptError != nil {
		log.Fatalf("Err: %v", sptError)
	}
	token := spt.Token()

	return token.AccessToken
}
