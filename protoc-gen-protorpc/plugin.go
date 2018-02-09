// Copyright 2017 <chaishushan{AT}gmail.com>. All rights reserved.
// Use of this source code is governed by a Apache
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"go/format"
	"log"
	"path"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	plugin "github.com/golang/protobuf/protoc-gen-go/plugin"
)

// mainPlugin produce the Service interface.
type mainPlugin struct {
	*generator.Generator
	CodeGenerator
}

// Name returns the name of the plugin.
func (p *mainPlugin) Name() string { return "main-plugin" }

// Init is called once after data structures are built but before
// code generation begins.
func (p *mainPlugin) Init(g *generator.Generator) {
	p.Generator = g
}

func (p *mainPlugin) InitService(g CodeGenerator) {
	p.CodeGenerator = g
}

// Generate produces the code generated by the plugin for this file.
func (p *mainPlugin) GenerateImports(file *generator.FileDescriptor) {
	// skip
}

// Generate generates the Service interface.
// rpc service can't handle other proto message!!!
func (p *mainPlugin) Generate(file *generator.FileDescriptor) {
	if !p.isFileNeedGenerate(file) {
		return
	}

	var buf bytes.Buffer
	fmt.Fprintln(&buf, p.HeaderCode(p.Generator, file))

	for _, msg := range file.MessageType {
		fmt.Fprintln(&buf, p.MessageCode(p.Generator, file, msg))
	}

	for _, svc := range file.Service {
		fmt.Fprintln(&buf, p.ServiceCode(p.Generator, file, svc))
	}

	fileContent := buf.String()
	if code, err := format.Source(buf.Bytes()); err != nil {
		log.Printf("mainPlugin.Generate: format %q failed, err = %v", file.GetName(), err)
	} else {
		fileContent = string(code)
	}

	p.Generator.Response.File = append(p.Generator.Response.File, &plugin.CodeGeneratorResponse_File{
		Name:    proto.String(p.goFileName(file)),
		Content: proto.String(fileContent),
	})
}

func (p *mainPlugin) isFileNeedGenerate(file *generator.FileDescriptor) bool {
	for _, v := range p.Generator.Request.FileToGenerate {
		if v == file.GetName() {
			return true
		}
	}
	return false
}

func (p *mainPlugin) goFileName(file *generator.FileDescriptor) string {
	name := *file.Name
	if ext := path.Ext(name); ext == ".proto" || ext == ".protodevel" {
		name = name[:len(name)-len(ext)]
	}
	name += p.FileNameExt()

	// Does the file have a "go_package" option?
	// If it does, it may override the filename.
	if impPath, _, ok := p.goPackageOption(file); ok && impPath != "" {
		// Replace the existing dirname with the declared import path.
		_, name = path.Split(name)
		name = path.Join(impPath, name)
		return name
	}

	return name
}

func (p *mainPlugin) goPackageOption(file *generator.FileDescriptor) (impPath, pkg string, ok bool) {
	pkg = file.GetOptions().GetGoPackage()
	if pkg == "" {
		return
	}
	ok = true
	// The presence of a slash implies there's an import path.
	slash := strings.LastIndex(pkg, "/")
	if slash < 0 {
		return
	}
	impPath, pkg = pkg, pkg[slash+1:]
	// A semicolon-delimited suffix overrides the package name.
	sc := strings.IndexByte(impPath, ';')
	if sc < 0 {
		return
	}
	impPath, pkg = impPath[:sc], impPath[sc+1:]
	return
}

var pkgCodeGeneratorList []CodeGenerator

type CodeGenerator interface {
	Name() string
	FileNameExt() string

	HeaderCode(g *generator.Generator, file *generator.FileDescriptor) string
	ServiceCode(p *generator.Generator, file *generator.FileDescriptor, svc *descriptor.ServiceDescriptorProto) string
	MessageCode(p *generator.Generator, file *generator.FileDescriptor, msg *descriptor.DescriptorProto) string
}

func RegisterCodeGenerator(g CodeGenerator) {
	pkgCodeGeneratorList = append(pkgCodeGeneratorList, g)
}

func getAllCodeGenerator() []CodeGenerator {
	return pkgCodeGeneratorList
}

func getAllServiceGeneratorNames() (names []string) {
	for _, g := range pkgCodeGeneratorList {
		names = append(names, g.Name())
	}
	return
}

func getFirstServiceGeneratorName() string {
	if len(pkgCodeGeneratorList) > 0 {
		return pkgCodeGeneratorList[0].Name()
	}
	return ""
}

func getCodeGenerator(name string) CodeGenerator {
	for _, g := range pkgCodeGeneratorList {
		if g.Name() == name {
			return g
		}
	}
	return nil
}
