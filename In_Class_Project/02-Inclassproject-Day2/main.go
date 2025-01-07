package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
)

type InputData struct {
	People []Person
}

type Person struct {
	Name      string
	Age       int
	Salary    float64
	Education string
}

type Stats struct {
	AverageAge      float64
	YoungestPeople  []string
	OldestPeople    []string
	HighestSalary   []string
	LowestSalary    []string
	EducationCounts map[string]int
}

func main() {
	data, err := os.ReadFile("./people.json")
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	var input InputData
	if err := json.Unmarshal(data, &input); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return
	}

	stats := calculateStats(input.People)

	outputJSON, err := json.MarshalIndent(stats, "", "    ")
	if err != nil {
		fmt.Printf("Error creating JSON output: %v\n", err)
		return
	}

	err = os.WriteFile("output.json", outputJSON, 0644)
	if err != nil {
		fmt.Printf("Error writing output file: %v\n", err)
		return
	}
}

func calculateStats(people []Person) Stats {
	stats := Stats{
		EducationCounts: make(map[string]int),
	}

	if len(people) == 0 {
		return stats
	}

	minAge := people[0].Age
	maxAge := people[0].Age
	minSalary := people[0].Salary
	maxSalary := people[0].Salary
	totalAge := 0

	for _, person := range people {
		totalAge += person.Age

		if person.Age < minAge {
			minAge = person.Age
			stats.YoungestPeople = []string{person.Name}
		} else if person.Age == minAge {
			stats.YoungestPeople = append(stats.YoungestPeople, person.Name)
		}

		if person.Age > maxAge {
			maxAge = person.Age
			stats.OldestPeople = []string{person.Name}
		} else if person.Age == maxAge {
			stats.OldestPeople = append(stats.OldestPeople, person.Name)
		}

		if person.Salary > maxSalary {
			maxSalary = person.Salary
			stats.HighestSalary = []string{person.Name}
		} else if math.Abs(person.Salary-maxSalary) < 0.01 {
			stats.HighestSalary = append(stats.HighestSalary, person.Name)
		}

		if person.Salary < minSalary {
			minSalary = person.Salary
			stats.LowestSalary = []string{person.Name}
		} else if math.Abs(person.Salary-minSalary) < 0.01 {
			stats.LowestSalary = append(stats.LowestSalary, person.Name)
		}

		stats.EducationCounts[person.Education]++
	}

	stats.AverageAge = float64(totalAge) / float64(len(people))

	return stats
}
