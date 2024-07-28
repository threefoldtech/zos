package rpc

import (
	"encoding/json"

	"github.com/threefoldtech/zos/pkg/capacity/dmi"
)

// This function heavily used and it depends on matching json tags on both types
func convert(input any, output any) error {
	data, err := json.Marshal(input)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &output); err != nil {
		return err
	}

	return nil
}

func convertDmi(input *dmi.DMI, output *DMI) {
	dmi := *input

	*output = DMI{
		Tooling: Tooling{
			Aggregator: dmi.Tooling.Aggregator,
			Decoder:    dmi.Tooling.Decoder,
		},
		Sections: func() []Section {
			var sections []Section
			for _, sec := range dmi.Sections {
				section := Section{
					HandleLine: sec.HandleLine,
					TypeStr:    sec.TypeStr,
					Type:       uint64(sec.Type),
					SubSections: func() []SubSection {
						var subsections []SubSection
						for _, subsec := range sec.SubSections {
							subsection := SubSection{
								Title: subsec.Title,
								Properties: func() []PropertyData {
									var properties []PropertyData
									for key, prop := range subsec.Properties {
										property := PropertyData{
											Name:  key,
											Val:   prop.Val,
											Items: prop.Items,
										}
										properties = append(properties, property)
									}
									return properties
								}(),
							}
							subsections = append(subsections, subsection)
						}
						return subsections
					}(),
				}
				sections = append(sections, section)
			}
			return sections
		}(),
	}
}
