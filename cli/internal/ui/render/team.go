package render

import (
	"aide/cli/internal/persistence/store"
	"fmt"
	"strings"
)

func PrintTeamList(members []store.Member, view string) {
	if view == "flat" {
		printTeamFlat(members)
	} else {
		printTeamTree(members)
	}
}

func printTeamFlat(members []store.Member) {
	byID := make(map[int64]store.Member, len(members))
	for _, m := range members {
		byID[m.ID] = m
	}

	w := newTabWriter()
	fmt.Fprintf(w, "  ID\tNAME\tREGISTRATION\tROLE\tDEPARTMENT\tMANAGER\n")
	fmt.Fprintf(w, "  --\t----\t------------\t----\t----------\t-------\n")
	for _, m := range members {
		managerName := "—"
		if m.ManagerID != nil {
			if mgr, ok := byID[*m.ManagerID]; ok {
				managerName = mgr.Name
			}
		}
		fmt.Fprintf(w, "  %d\t%s\t%s\t%s\t%s\t%s\n",
			m.ID, m.Name, m.Registration, m.Role, m.Department, managerName)
	}
	w.Flush()
}

func printTeamTree(members []store.Member) {
	children := make(map[int64][]store.Member)
	var roots []store.Member

	for _, m := range members {
		if m.ManagerID == nil {
			roots = append(roots, m)
		} else {
			children[*m.ManagerID] = append(children[*m.ManagerID], m)
		}
	}

	var walk func(m store.Member, depth int)
	walk = func(m store.Member, depth int) {
		indent := strings.Repeat("  ", depth)
		line := fmt.Sprintf("%s%s", indent, m.Name)
		if m.Registration != "" {
			line += fmt.Sprintf(" (%s)", m.Registration)
		}
		if m.Role != "" {
			line += " — " + m.Role
		}
		fprintln(line)
		for _, child := range children[m.ID] {
			walk(child, depth+1)
		}
	}

	for _, root := range roots {
		walk(root, 0)
	}
}

func FormatTeamTree(members []store.Member) string {
	children := make(map[int64][]store.Member)
	var roots []store.Member

	for _, m := range members {
		if m.ManagerID == nil {
			roots = append(roots, m)
		} else {
			children[*m.ManagerID] = append(children[*m.ManagerID], m)
		}
	}

	var sb strings.Builder

	var walk func(m store.Member, depth int)
	walk = func(m store.Member, depth int) {
		indent := strings.Repeat("  ", depth)
		line := fmt.Sprintf("%s%s", indent, m.Name)
		if m.Registration != "" {
			line += fmt.Sprintf(" (%s)", m.Registration)
		}
		if m.Role != "" {
			line += " — " + m.Role
		}
		sb.WriteString(line + "\n")
		for _, child := range children[m.ID] {
			walk(child, depth+1)
		}
	}

	for _, root := range roots {
		walk(root, 0)
	}

	return strings.TrimRight(sb.String(), "\n")
}
