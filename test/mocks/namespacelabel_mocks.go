package mocks

import (
	labelsv1 "github.com/matanamar10/namespacelabel-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "os"
)

var (
	NamespaceName      = "example-namespace"
	NamespacelabelName = "namespacelabel-sample"
)

func CreateBaseNamespacelabel() *labelsv1.Namespacelabel {
	return &labelsv1.Namespacelabel{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespacelabel",
			APIVersion: "labels.dana.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      NamespacelabelName,
			Namespace: NamespaceName,
		},
		Spec: labelsv1.NamespacelabelSpec{
			Labels: map[string]string{
				"environment": "test",
				"owner":       "team-alpha",
			},
		},
	}
}

func CreateBaseNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: NamespaceName,
		},
	}
}

func CreateCustomNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func CreateCustomNamespacelabel(labels map[string]string) *labelsv1.Namespacelabel {
	return &labelsv1.Namespacelabel{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespacelabel",
			APIVersion: "labels.dana.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      NamespacelabelName,
			Namespace: NamespaceName,
		},
		Spec: labelsv1.NamespacelabelSpec{
			Labels: labels,
		},
	}
}
