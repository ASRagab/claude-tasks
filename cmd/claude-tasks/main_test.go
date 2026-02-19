package main

import "testing"

func TestParseTUISchedulerMode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    tuiSchedulerMode
		wantErr bool
	}{
		{name: "auto", input: "auto", want: tuiSchedulerAuto},
		{name: "on", input: "on", want: tuiSchedulerOn},
		{name: "off", input: "off", want: tuiSchedulerOff},
		{name: "case insensitive", input: "On", want: tuiSchedulerOn},
		{name: "trim spaces", input: " off ", want: tuiSchedulerOff},
		{name: "invalid", input: "maybe", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTUISchedulerMode(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for input %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("parse mode: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestShouldStartTUIScheduler(t *testing.T) {
	tests := []struct {
		name         string
		mode         tuiSchedulerMode
		daemonActive bool
		want         bool
	}{
		{name: "auto with daemon running", mode: tuiSchedulerAuto, daemonActive: true, want: false},
		{name: "auto without daemon", mode: tuiSchedulerAuto, daemonActive: false, want: true},
		{name: "on with daemon running", mode: tuiSchedulerOn, daemonActive: true, want: true},
		{name: "off without daemon", mode: tuiSchedulerOff, daemonActive: false, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldStartTUIScheduler(tt.mode, tt.daemonActive)
			if got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
