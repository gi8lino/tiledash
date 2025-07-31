package templates

import "html/template"

// TemplateFuncMap returns all helper functions for templates.
func TemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"add":        templateAdd,
		"append":     templateAppend,
		"dict":       templateDict,
		"dig":        templateDig,
		"formatDate": formatDate,
		"keys":       templateKeys,
		"list":       templateList,
		"listany":    templateListAny,
		"set":        templateSet,
		"slice":      templateSlice,
	}
}
