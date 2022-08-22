package main

import (
	"fmt"
	"io/ioutil"

	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/walk"
)

type Visitor struct{}

func (m Visitor) VisitMap(nodes walk.Sources, s *openapi.ResourceSchema) (*yaml.RNode, error) {
	// fmt.Println(nodes.String())
	return nodes.Dest(), nil
}

func (m Visitor) VisitScalar(nodes walk.Sources, _ *openapi.ResourceSchema) (*yaml.RNode, error) {
	fmt.Println(nodes.String())
	// nodes.Dest().Field()
	return nodes.Dest(), nil
}

func (m Visitor) VisitList(nodes walk.Sources, _ *openapi.ResourceSchema, _ walk.ListKind) (*yaml.RNode, error) {
	// fmt.Println(nodes.String())
	return nodes.Dest(), nil
}

func main() {
	x := kio.LocalPackageReader{
		PackagePath:           "/Users/yexiaodong/go/src/github.com/yeahdongcn/kustohelmize/test/testdata/",
		OmitReaderAnnotations: true,
	}
	nodes, err := x.Read()
	if err != nil {
		panic(err)
	}
	for _, n := range nodes {
		zz := walk.Walker{
			Visitor:            Visitor{},
			VisitKeysAsScalars: true,
			Sources:            []*yaml.RNode{n},
		}
		zz.Walk()
	}
	fmt.Println(nodes)

	node, err := kyaml.ReadFile("/Users/yexiaodong/go/src/github.com/yeahdongcn/kustohelmize/test/testdata/0200_sample.yaml")
	if err != nil {
		panic(err)
	}
	fmt.Println(node)
	_, err = ioutil.ReadFile("/Users/yexiaodong/go/src/github.com/yeahdongcn/kustohelmize/test/testdata/0200_sample.yaml")
	if err != nil {
		panic(err)
	}
}
