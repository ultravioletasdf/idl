package compile_go

import (
	"fmt"
	"idl/parser"
	"os"
	"path"
)

type Compiler struct {
	tree     parser.Nodes
	filename string
	file     *os.File
}

func New(filename string, tree parser.Nodes) *Compiler {
	return &Compiler{tree: tree, filename: filename}
}
func (c *Compiler) Close() error {
	return c.file.Close()
}
func (c *Compiler) Compile() {
	fmt.Println("Compiling for go...")
	out := c.tree.Package
	optionOut := c.option("go_out")
	if optionOut != "" {
		out = optionOut
	}
	err := os.MkdirAll(out, os.ModePerm)
	if err != nil {
		panic(err)
	}
	file, err := os.Create(path.Join(out, c.filename+".idl.go"))
	if err != nil {
		panic(err)
	}
	c.file = file
	c.compileStructs()
}
func (c *Compiler) option(name string) string {
	for opt := range c.tree.Options {
		if c.tree.Options[opt].Name == name {
			return c.tree.Options[opt].Value
		}
	}
	return ""
}

func (c *Compiler) compileStructs() {
	_, err := fmt.Fprintf(c.file, "package %s\n\nimport \"encoding/binary\"\n\n", c.tree.Package)
	if err != nil {
		panic(err)
	}
	for _, structure := range c.tree.Structures {
		data := fmt.Sprintf("type %s struct {\n", structure.Name)
		for _, field := range structure.Fields {
			if field.Type == "int" {
				field.Type = "int64"
			}
			data += fmt.Sprintf("\t%s %s\n", field.Name, field.Type)
		}
		data += fmt.Sprintf("}\nfunc (d *%s) Encode() ([]byte, error) {\n\tvar bin []byte\n\tvar err error\n", structure.Name)
		for _, field := range structure.Fields {
			if field.Type == "string" {
				data += fmt.Sprintf("\t%s := make([]byte, 64)\n\tcopy(%s[:], []byte(d.%s))\n\tbin, err = binary.Append(bin, binary.LittleEndian, %s)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n", field.Name, field.Name, field.Name, field.Name)
			} else {
				data += fmt.Sprintf("\tbin, err = binary.Append(bin, binary.LittleEndian, d.%s)\n\tif err != nil {\n\t\treturn nil, err\n\t}\n", field.Name)
			}
		}
		data += "\treturn bin, nil\n}\n"

		data += fmt.Sprintf("\nfunc (d *%s) Decode(bin []byte) {\n", structure.Name)
		var offset int
		for _, field := range structure.Fields {
			switch field.Type {
			case "string":
				data += fmt.Sprintf("\td.%s = string(bin[%d:%d])\n", field.Name, offset, offset+64)
				offset += 64
			case "int64":
				data += fmt.Sprintf("\td.%s = int64(binary.LittleEndian.Uint64(bin[%d:%d]))\n", field.Name, offset, offset+8)
				offset += 8
			case "int32":
				data += fmt.Sprintf("\td.%s = int32(binary.LittleEndian.Uint32(bin[%d:%d]))\n", field.Name, offset, offset+4)
				offset += 4
			case "int16":
				data += fmt.Sprintf("\td.%s = int16(binary.LittleEndian.Uint16(bin[%d:%d]))\n", field.Name, offset, offset+2)
				offset += 2
			case "int8":
				data += fmt.Sprintf("\td.%s = int8(bin[%d])\n", field.Name, offset)
				offset += 1
			}
		}
		data += "}\n"

		_, err := fmt.Fprint(c.file, data)
		if err != nil {
			panic(err)
		}
	}
}
