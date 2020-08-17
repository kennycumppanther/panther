package mage

/**
 * Panther is a Cloud-Native SIEM for the Modern Security Team.
 * Copyright (C) 2020 Panther Labs Inc
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

import (
	"bytes"
	"fmt"
	"html"
	"math"
	"path/filepath"
	"sort"
	"strings"

	"github.com/panther-labs/panther/internal/log_analysis/awsglue"
	"github.com/panther-labs/panther/internal/log_analysis/log_processor/registry"
	"github.com/panther-labs/panther/tools/cfndoc"
	"github.com/panther-labs/panther/tools/config"
)

// Preview auto-generated documentation in out/doc
func Doc() {
	if err := doc(); err != nil {
		log.Fatal(err)
	}
	log.Info("doc: generated runbooks and log types in out/docs")
}

func doc() error {
	if err := opDocs(); err != nil {
		return err
	}
	return logDocs()
}

const (
	// paths are relative to docs/gitbook/operations/runbooks.md

	inventoryDocHeader = `	
<!-- This document is generated by "mage doc". DO NOT EDIT! -->	
# Panther Application Run Books	
Refer to the 	
[Cloud Security](../cloud-security/README.md)	
and	
[Log Analysis](../log-analysis/README.md)	
architecture diagrams for context.	
Resource names below refer to resources in the Cloud Formation templates in Panther.	
Each resource describes its function and failure impacts.	
`
)

// Return the list of Panther's CloudFormation files
func cfnFiles() []string {
	paths, err := filepath.Glob("deployments/*.yml")
	if err != nil {
		log.Fatalf("failed to glob deployments: %v", err)
	}

	// Remove the config file
	var result []string
	for _, p := range paths {
		if p != config.Filepath {
			result = append(result, p)
		}
	}
	return result
}

// generate operational documentation from deployment CloudFormation
func opDocs() error {
	log.Debug("doc: generating operational documentation from cloudformation")
	docs, err := cfndoc.ReadCfn(cfnFiles()...)
	if err != nil {
		return fmt.Errorf("failed to generate operational documentation: %v", err)
	}

	var docsBuffer bytes.Buffer
	docsBuffer.WriteString(inventoryDocHeader)
	for _, doc := range docs {
		docsBuffer.WriteString(fmt.Sprintf("## %s\n%s\n\n", doc.Resource, doc.Documentation))
	}

	return writeFile(filepath.Join("out", "docs", "gitbook", "operations", "runbooks.md"), docsBuffer.Bytes())
}

const (
	parserReadmeHeader = `	
<!-- This document is generated by "mage doc". DO NOT EDIT! -->	
`
)

type supportedLogs struct {
	Categories map[string]*logCategory
	TotalTypes int
}

// Generate entire "supported-logs" documentation directory
func (logs *supportedLogs) generateDocumentation() error {
	outDir := filepath.Join("out", "docs", "gitbook", "log-analysis", "log-processing", "supported-logs")

	// Write one file for each category.
	for _, category := range logs.Categories {
		if err := category.generateDocFile(outDir); err != nil {
			return err
		}
	}

	return nil
}

type logCategory struct {
	Name     string
	LogTypes []string
}

// Generate a single documentation file for a log category, e.g. "AWS.md"
func (category *logCategory) generateDocFile(outDir string) error {
	sort.Strings(category.LogTypes)

	var docsBuffer bytes.Buffer
	docsBuffer.WriteString(parserReadmeHeader)
	docsBuffer.WriteString(fmt.Sprintf("# %s\n%sRequired fields are in <b>bold</b>.%s\n",
		category.Name,
		`{% hint style="info" %}`,
		`{% endhint %}`))

	// use html table to get needed control
	for _, logType := range category.LogTypes {
		entry := registry.Lookup(logType)
		table := entry.GlueTableMeta()
		entryDesc := entry.Describe()
		desc := entryDesc.Description
		if entryDesc.ReferenceURL != "-" {
			desc += "\n" + "Reference: " + entryDesc.ReferenceURL + "\n"
		}

		description := html.EscapeString(desc)

		docsBuffer.WriteString(fmt.Sprintf("## %s\n%s\n", logType, description))

		// add schema as html table since markdown won't let you embed tables
		docsBuffer.WriteString(`<table>` + "\n")
		docsBuffer.WriteString("<tr><th align=center>Column</th><th align=center>Type</th><th align=center>Description</th></tr>\n") // nolint

		columns, _ := awsglue.InferJSONColumns(table.EventStruct(), awsglue.GlueMappings...) // get the Glue schema
		for _, column := range columns {
			colName := column.Name
			if column.Required {
				colName = "<b>" + colName + "</b>" // required elements are bold
			}
			docsBuffer.WriteString(fmt.Sprintf("<tr><td valign=top>%s</td><td>%s</td><td valign=top>%s</td></tr>\n",
				formatColumnName(colName),
				formatType(logType, column),
				html.EscapeString(column.Comment)))
		}

		docsBuffer.WriteString("</table>\n\n")
	}

	path := filepath.Join(outDir, category.Name+".md")
	log.Debugf("writing log category documentation: %s", path)
	return writeFile(path, docsBuffer.Bytes())
}

func logDocs() error {
	log.Debug("doc: generating documentation on supported logs")

	// allow large comment descriptions in the docs (by default they are clipped)
	awsglue.MaxCommentLength = math.MaxInt32
	defer func() {
		awsglue.MaxCommentLength = awsglue.DefaultMaxCommentLength
	}()

	logs, err := findSupportedLogs()
	if err != nil {
		return err
	}

	return logs.generateDocumentation()
}

// Group log registry by category
func findSupportedLogs() (*supportedLogs, error) {
	result := supportedLogs{Categories: make(map[string]*logCategory)}

	tables := registry.AvailableTables()
	for _, table := range tables {
		logType := table.LogType()
		categoryType := strings.Split(logType, ".")
		if len(categoryType) != 2 {
			return nil, fmt.Errorf("unexpected logType format: %s", logType)
		}
		name := categoryType[0]

		category, exists := result.Categories[name]
		if !exists {
			category = &logCategory{Name: name}
			result.Categories[name] = category
		}
		category.LogTypes = append(category.LogTypes, logType)
		result.TotalTypes++
	}

	return &result, nil
}

func formatColumnName(name string) string {
	return "<code>" + name + "</code>"
}

func formatType(logType string, col awsglue.Column) string {
	return "<code>" + prettyPrintType(logType, col.Name, col.Type, "") + "</code>"
}

const (
	prettyPrintPrefix = "<br>"
	prettyPrintIndent = "&nbsp;&nbsp;"
)

func prettyPrintType(logType, colName, colType, indent string) string {
	complexTypes := []string{"array", "struct", "map"}
	for _, ct := range complexTypes {
		if strings.HasPrefix(colType, ct) {
			return prettyPrintComplexType(logType, colName, ct, colType, indent)
		}
	}

	// if NOT a complex type we just use the Glue type
	return colType
}

// complex hive types are ugly
func prettyPrintComplexType(logType, colName, complexType, colType, indent string) (pretty string) {
	switch complexType {
	case "array":
		return prettyPrintArrayType(logType, colName, colType, indent)
	case "map":
		return prettyPrintMapType(logType, colName, colType, indent)
	case "struct":
		return prettyPrintStructType(logType, colName, colType, indent)
	default:
		panic("unknown complex type: " + complexType + " for " + colName + " in " + logType)
	}
}

func prettyPrintArrayType(logType, colName, colType, indent string) string {
	fields := getTypeFields("array", colType)
	if len(fields) != 1 {
		panic("could not parse array type `" + colType + "` for " + colName + " in " + logType)
	}
	return "[" + prettyPrintType(logType, colName, fields[0], indent) + "]"
}

func prettyPrintMapType(logType, colName, colType, indent string) string {
	fields := getTypeFields("map", colType)
	if len(fields) != 2 {
		panic("could not parse map type `" + colType + "` for " + colName + " in " + logType)
	}
	keyType := fields[0]
	valType := fields[1]
	indent += prettyPrintIndent
	return "{" + prettyPrintPrefix + indent + prettyPrintType(logType, colName, keyType, indent) + ":" +
		prettyPrintType(logType, colName, valType, indent) + prettyPrintPrefix + "}"
}

func prettyPrintStructType(logType, colName, colType, indent string) string {
	fields := getTypeFields("struct", colType)
	if len(fields) == 0 {
		panic("could not parse struct type `" + colType + "` for " + colName + " in " + logType)
	}
	indent += prettyPrintIndent
	var fieldTypes []string
	for _, field := range fields {
		splitIndex := strings.Index(field, ":") // name:type (can't use Split() cuz type can have ':'
		if splitIndex == -1 {
			panic("could not parse struct field `" + field + "` of `" + colType + "` for " + colName + " in " + logType)
		}
		name := `"` + field[0:splitIndex] + `"` // make it look like JSON by quoting
		structFieldType := field[splitIndex+1:]
		fieldTypes = append(fieldTypes, prettyPrintPrefix+indent+name+":"+
			prettyPrintType(logType, colName, structFieldType, indent))
	}
	return "{" + strings.Join(fieldTypes, ",") + prettyPrintPrefix + "}"
}

func getTypeFields(complexType, colType string) (subFields []string) {
	// strip off complexType + '<' in front and '>' on end
	fields := colType[len(complexType)+1 : len(colType)-1]
	// split fields into subFields around top level commas in type definition
	startSubfieldIndex := 0
	insideBracketCount := 0 // when non-zero we are inside a complex type
	var index int
	for index = range fields {
		if fields[index] == ',' && insideBracketCount == 0 {
			subFields = append(subFields, fields[startSubfieldIndex:index])
			startSubfieldIndex = index + 1 // next
		}
		// track context
		if fields[index] == '<' {
			insideBracketCount++
		} else if fields[index] == '>' {
			insideBracketCount--
		}
	}
	if len(fields[startSubfieldIndex:]) > 0 { // the rest
		subFields = append(subFields, fields[startSubfieldIndex:])
	}
	return subFields
}
