package vm

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/tetrafolium/goby/vm/classes"
	"github.com/tetrafolium/goby/vm/errors"
)

// Instance methods --------------------------------------------------------

func builtinHTTPClientInstanceMethods() []*BuiltinMethodObject {
	//TODO: cookie jar and mutable client
	goClient := http.DefaultClient

	return []*BuiltinMethodObject{
		{
			// Sends a GET request to the target and returns a `Net::HTTP::Response` object.
			Name: "get",
			Fn: func(receiver Object, sourceLine int, t *Thread, args []Object, blockFrame *normalCallFrame) Object {
				if len(args) != 1 {
					return t.vm.InitErrorObject(errors.ArgumentError, sourceLine, errors.WrongNumberOfArgument, 1, len(args))
				}

				typeErr := t.vm.checkArgTypes(args, sourceLine, classes.StringClass)

				if typeErr != nil {
					return typeErr
				}

				resp, err := goClient.Get(args[0].Value().(string))
				if err != nil {
					return t.vm.InitErrorObject(errors.HTTPError, sourceLine, couldNotCompleteRequest, err)
				}

				gobyResp, err := responseGoToGoby(t, resp)
				if err != nil {
					return t.vm.InitErrorObject(errors.InternalError, sourceLine, err.Error())
				}

				return gobyResp

			},
		}, {
			// Sends a POST request to the target and returns a `Net::HTTP::Response` object.
			Name: "post",
			Fn: func(receiver Object, sourceLine int, t *Thread, args []Object, blockFrame *normalCallFrame) Object {
				if len(args) != 3 {
					return t.vm.InitErrorObject(errors.ArgumentError, sourceLine, errors.WrongNumberOfArgument, 3, len(args))
				}

				typeErr := t.vm.checkArgTypes(args, sourceLine, classes.StringClass, classes.StringClass, classes.StringClass)

				if typeErr != nil {
					return typeErr
				}

				bodyR := strings.NewReader(args[2].Value().(string))

				resp, err := goClient.Post(args[0].Value().(string), args[1].Value().(string), bodyR)
				if err != nil {
					return t.vm.InitErrorObject(errors.HTTPError, sourceLine, "Could not complete request, %s", err)
				}

				gobyResp, err := responseGoToGoby(t, resp)
				if err != nil {
					return t.vm.InitErrorObject(errors.InternalError, sourceLine, err.Error())
				}

				return gobyResp

			},
		}, {
			// Sends a HEAD request to the target and returns a `Net::HTTP::Response` object.
			Name: "head",
			Fn: func(receiver Object, sourceLine int, t *Thread, args []Object, blockFrame *normalCallFrame) Object {
				if len(args) != 1 {
					return t.vm.InitErrorObject(errors.ArgumentError, sourceLine, errors.WrongNumberOfArgument, 1, len(args))
				}

				typeErr := t.vm.checkArgTypes(args, sourceLine, classes.StringClass)

				if typeErr != nil {
					return typeErr
				}

				resp, err := goClient.Head(args[0].Value().(string))
				if err != nil {
					return t.vm.InitErrorObject(errors.HTTPError, sourceLine, couldNotCompleteRequest, err)
				}

				gobyResp, err := responseGoToGoby(t, resp)
				if err != nil {
					return t.vm.InitErrorObject(errors.InternalError, sourceLine, err.Error())
				}

				return gobyResp

			},
		}, {
			// Returns a blank `Net::HTTP::Request` object to be sent with the`exec` method
			Name: "request",
			Fn: func(receiver Object, sourceLine int, t *Thread, args []Object, blockFrame *normalCallFrame) Object {
				return httpRequestClass.initializeInstance()

			},
		}, {
			// Sends a passed `Net::HTTP::Request` object and returns a `Net::HTTP::Response` object
			Name: "exec",
			Fn: func(receiver Object, sourceLine int, t *Thread, args []Object, blockFrame *normalCallFrame) Object {
				if len(args) != 1 {
					return t.vm.InitErrorObject(errors.ArgumentError, sourceLine, errors.WrongNumberOfArgument, 1, len(args))
				}

				typeErr := t.vm.checkArgTypes(args, sourceLine, httpRequestClass.Name)

				if typeErr != nil {
					return typeErr
				}

				goReq, err := requestGobyToGo(args[0])
				if err != nil {
					return t.vm.InitErrorObject(errors.ArgumentError, sourceLine, err.Error())
				}

				goResp, err := goClient.Do(goReq)
				if err != nil {
					return t.vm.InitErrorObject(errors.HTTPError, sourceLine, couldNotCompleteRequest, err)
				}

				gobyResp, err := responseGoToGoby(t, goResp)

				if err != nil {
					return t.vm.InitErrorObject(errors.InternalError, sourceLine, err.Error())
				}

				return gobyResp

			},
		},
	}
}

// Internal functions ===================================================

// Functions for initialization -----------------------------------------

func initClientClass(vm *VM, hc *RClass) *RClass {
	clientClass := vm.initializeClass("Client")
	hc.setClassConstant(clientClass)

	clientClass.setBuiltinMethods(builtinHTTPClientInstanceMethods(), false)

	httpClientClass = clientClass
	return clientClass
}

// Other helper functions -----------------------------------------------

func requestGobyToGo(gobyReq Object) (*http.Request, error) {
	//:method, :protocol, :body, :content_length, :transfer_encoding, :host, :path, :url, :params
	uObj, ok := gobyReq.InstanceVariableGet("@url")
	if !ok {
		return nil, fmt.Errorf("could not get url")
	}

	u := uObj.(*StringObject).value

	methodObj, ok := gobyReq.InstanceVariableGet("@method")
	if !ok {
		return nil, fmt.Errorf("could not get method")
	}

	method := methodObj.(*StringObject).value

	var body string
	if !(method == "GET" || method == "HEAD") {
		bodyObj, ok := gobyReq.InstanceVariableGet("@body")
		if !ok {
			return nil, fmt.Errorf("could not get body")
		}

		body = bodyObj.(*StringObject).value
	}

	return http.NewRequest(method, u, strings.NewReader(body))

}

func responseGoToGoby(t *Thread, goResp *http.Response) (Object, error) {
	gobyResp := httpResponseClass.initializeInstance()

	//attr_accessor :body, :status, :status_code, :protocol, :transfer_encoding, :http_version, :request_http_version, :request
	//attr_reader :headers

	body, err := ioutil.ReadAll(goResp.Body)
	if err != nil {
		return nil, err
	}

	gobyResp.InstanceVariableSet("@body", t.vm.InitStringObject(string(body)))
	gobyResp.InstanceVariableSet("@status_code", t.vm.InitObjectFromGoType(goResp.StatusCode))
	gobyResp.InstanceVariableSet("@status", t.vm.InitObjectFromGoType(goResp.Status))
	gobyResp.InstanceVariableSet("@protocol", t.vm.InitObjectFromGoType(goResp.Proto))
	gobyResp.InstanceVariableSet("@transfer_encoding", t.vm.InitObjectFromGoType(goResp.TransferEncoding))

	underHeaders := map[string]Object{}

	for k, v := range goResp.Header {
		underHeaders[k] = t.vm.InitObjectFromGoType(v)
	}

	gobyResp.InstanceVariableSet("@headers", t.vm.InitHashObject(underHeaders))

	return gobyResp, nil
}
