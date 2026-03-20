package dquely_test

import (
	"github.com/vibros68/dquely"
	"testing"
)

type Employee struct {
	Uid           string          `json:"uid,omitempty" dquely:"uid"`
	Name          string          `json:"name,omitempty" dquely:"name"`
	Bio           string          `json:"bio,omitempty" dquely:"bio"`
	CurrentJob    *EmployeeJob    `json:"currentJob,omitempty" dquely:"currentJob"`
	CurrentStatus *EmployeeStatus `json:"currentStatus,omitempty" dquely:"currentStatus"`
	CurrentSalary *EmployeeSalary `json:"currentSalary,omitempty" dquely:"currentSalary"`
	InCompany     *Company        `json:"inCompany,omitempty" dquely:"inCompany"`
}

type EmployeeStatus struct {
	Uid         string    `json:"uid,omitempty" dquely:"uid"`
	ForEmployee *Employee `json:"forEmployee,omitempty" dquely:"forEmployee"`
}

type EmployeeSalary struct {
	Uid         string    `json:"uid,omitempty" dquely:"uid"`
	ForEmployee *Employee `json:"forEmployee,omitempty" dquely:"forEmployee"`
}

type EmployeeJob struct {
	Uid         string    `json:"uid,omitempty" dquely:"uid"`
	ForEmployee *Employee `json:"forEmployee,omitempty" dquely:"forEmployee"`
}

func TestSetUIDs(t *testing.T) {
	var employee = &Employee{
		Name:          "Alice",
		CurrentJob:    &EmployeeJob{},
		CurrentStatus: &EmployeeStatus{},
		CurrentSalary: &EmployeeSalary{},
	}
	var uids = map[string]string{
		"currentJob":    "0x13891",
		"currentSalary": "0x13893",
		"currentStatus": "0x13892",
		"employee":      "0x13890",
	}
	err := dquely.SetUIDs(employee, uids)
	if err != nil {
		t.Fatalf("expected err to be nil, got %v", err)
	}
	if employee.Uid != "0x13890" {
		t.Errorf("expected uid to be 0x13890, got %s", employee.Uid)
	}
	if employee.CurrentJob.Uid != "0x13891" {
		t.Errorf("expected uid to be 0x13891, got %s", employee.CurrentJob.Uid)
	}
	if employee.CurrentStatus.Uid != "0x13892" {
		t.Errorf("expected uid to be 0x13892, got %s", employee.CurrentStatus.Uid)
	}
	if employee.CurrentSalary.Uid != "0x13893" {
		t.Errorf("expected uid to be 0x13893, got %s", employee.CurrentSalary.Uid)
	}
}
