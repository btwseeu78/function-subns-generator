package main

import (
	"context"

	"github.com/hashicorp/packer-plugin-sdk/random"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/fieldpath"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	fnv1beta1 "github.com/crossplane/function-sdk-go/proto/v1beta1"
	"github.com/crossplane/function-sdk-go/request"
	"github.com/crossplane/function-sdk-go/resource"
	"github.com/crossplane/function-sdk-go/response"
	"github.com/crossplane/function-subns-generator/input/v1beta1"
)

// Function returns whatever response you ask it to.
type Function struct {
	fnv1beta1.UnimplementedFunctionRunnerServiceServer

	log logging.Logger
}

// RunFunction runs the Function.
func (f *Function) RunFunction(_ context.Context, req *fnv1beta1.RunFunctionRequest) (*fnv1beta1.RunFunctionResponse, error) {
	f.log.Info("Running function", "tag", req.GetMeta().GetTag())

	rsp := response.To(req, response.DefaultTTL)

	randomAlphaNum := random.AlphaNumLower(20)
	in := &v1beta1.RandomGen{}
	if err := request.GetInput(req, in); err != nil {
		response.Fatal(rsp, errors.Wrapf(err, "cannot get Function input from %T", req))
		return rsp, nil
	}

	// id := uuid2.New().String()

	// f.log.Info("Generated UUID FOR", "Observed", observed, "UUID", id)
	desired, err := request.GetDesiredComposedResources(req)

	if err != nil {
		response.Fatal(rsp, err)
		return rsp, nil
	}

	for _, obj := range in.Cfg.Objs {

		f.log.Info("DesiredComposed Resource", "Desired: ", desired, "Req: ", req.GetMeta().GetTag())

		observed, err := request.GetObservedComposedResources(req)

		if err != nil {
			response.Fatal(rsp, err)
			return rsp, nil
		}
		for _, fp := range obj.FieldPath {
			if observed[resource.Name(obj.Name)].Resource != nil {
				observedPaved, err := fieldpath.PaveObject(observed[resource.Name(obj.Name)].Resource)

				if err != nil {
					f.log.Info("Unable To Get The Required Field Path", "PavedData:", observedPaved, "FieldPath", obj.FieldPath)
					response.Fatal(rsp, err)
					return rsp, nil
				}

			}
			if observed[resource.Name(obj.Name)].Resource == nil {
				err := patchFieldValueToObject(fp, obj.Name+"-"+randomAlphaNum, desired[resource.Name(obj.Name)].Resource)

				if err != nil {
					response.Warning(rsp, err)
					f.log.Info("Unable To generate the unstructured conversion", "observed", observed[resource.Name(obj.Name)].Resource, "err", err)
					return rsp, nil
				}
			}

		}
	}
	err = response.SetDesiredComposedResources(rsp, desired)

	if err != nil {
		response.Warning(rsp, err)
		f.log.Info("Creating Desired resource failed", "desired", desired, "error", err)
		return rsp, nil
	}

	return rsp, nil
}

func patchFieldValueToObject(fpath string, id string, to runtime.Object) error {
	paved, err := fieldpath.PaveObject(to)
	if err != nil {
		return err
	}
	if err := paved.SetValue(fpath, id); err != nil {
		return err
	}
	return runtime.DefaultUnstructuredConverter.FromUnstructured(paved.UnstructuredContent(), to)
}
