package iam

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_userList_Len(t *testing.T) {
	tests := []struct {
		name string
		l    userList
		want int
	}{
		{
			name: "base",
			l: userList{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "Tom",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "Jack",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "Lisa",
					},
				},
			},
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.l.Len(); got != tt.want {
				t.Errorf("Len() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_userList_Swap(t *testing.T) {
	type args struct {
		i int
		j int
	}
	tests := []struct {
		name string
		l    userList
		args args
		want userList
	}{
		{
			name: "base",
			l: userList{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "Tom",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "Jack",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "Lisa",
					},
				},
			},
			args: args{
				i: 1,
				j: 2,
			},
			want: userList{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "Tom",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "Lisa",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "Jack",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.l.Swap(tt.args.i, tt.args.j)
		})
	}
}

func Test_userList_Less(t *testing.T) {
	now := time.Now()
	af := time.Now()
	type args struct {
		i int
		j int
	}
	tests := []struct {
		name string
		l    userList
		args args
		want bool
	}{
		{
			name: "base",
			l: userList{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "Tom",
						CreationTimestamp: metav1.Time{
							Time: now,
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "Jack",
						CreationTimestamp: metav1.Time{
							Time: af,
						},
					},
				},
			},
			args: args{
				i: 0,
				j: 1,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.l.Less(tt.args.i, tt.args.j); got != tt.want {
				t.Errorf("Less() = %v, want %v", got, tt.want)
			}
		})
	}
}
