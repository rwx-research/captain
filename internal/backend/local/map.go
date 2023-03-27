package local

import "gopkg.in/yaml.v3"

type Map struct {
	Order  []string
	Values map[string]string
}

func NewMapFromYAML(node yaml.Node) Map {
	order := make([]string, 0)
	values := make(map[string]string, 0)

	for i, subNode := range node.Content {
		if i%2 == 0 {
			order = append(order, subNode.Value)
		} else {
			values[order[len(order)-1]] = subNode.Value
		}
	}

	return Map{order, values}
}

func (m Map) Equals(n Map) bool {
	filteredM, strictM := m.withoutKey("strict")
	filteredN, strictN := n.withoutKey("strict")

	if strictM != strictN {
		return false
	}

	if len(filteredM.Order) != len(filteredN.Order) {
		return false
	}

	for i, key := range filteredM.Order {
		if key != filteredN.Order[i] {
			return false
		}

		if filteredM.Values[key] != filteredN.Values[key] {
			return false
		}
	}

	return true
}

func (m Map) ToYAML() yaml.Node {
	node := yaml.Node{
		Kind:    yaml.MappingNode,
		Content: make([]*yaml.Node, len(m.Order)*2),
	}

	for i, key := range m.Order {
		j := i * 2
		node.Content[j] = &yaml.Node{Kind: yaml.ScalarNode, Value: key}
		node.Content[j+1] = &yaml.Node{Kind: yaml.ScalarNode, Value: m.Values[key]}
	}

	return node
}

func (m Map) withoutKey(key string) (Map, string) {
	var value string

	filteredOrder := make([]string, 0)
	filteredValues := make(map[string]string)

	for _, element := range m.Order {
		if element != key {
			filteredOrder = append(filteredOrder, element)
			filteredValues[element] = m.Values[element]

			continue
		}

		if v, ok := m.Values[key]; ok {
			value = v
		}
	}

	return Map{filteredOrder, filteredValues}, value
}
