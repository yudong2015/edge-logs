{{/*
Expand the name of the chart.
*/}}
{{- define "edge-logs.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "edge-logs.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "edge-logs.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "edge-logs.labels" -}}
helm.sh/chart: {{ include "edge-logs.chart" . }}
{{ include "edge-logs.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "edge-logs.selectorLabels" -}}
app.kubernetes.io/name: {{ include "edge-logs.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "edge-logs.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "edge-logs.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
APIServer image
*/}}
{{- define "edge-logs.apiserver.image" -}}
{{- $registry := .Values.apiserver.image.registry | default .Values.global.imageRegistry -}}
{{- if $registry }}
{{- printf "%s/%s:%s" $registry .Values.apiserver.image.repository .Values.apiserver.image.tag }}
{{- else }}
{{- printf "%s:%s" .Values.apiserver.image.repository .Values.apiserver.image.tag }}
{{- end }}
{{- end }}

{{/*
Frontend image
*/}}
{{- define "edge-logs.frontend.image" -}}
{{- $registry := .Values.frontend.image.registry | default .Values.global.imageRegistry -}}
{{- if $registry }}
{{- printf "%s/%s:%s" $registry .Values.frontend.image.repository .Values.frontend.image.tag }}
{{- else }}
{{- printf "%s:%s" .Values.frontend.image.repository .Values.frontend.image.tag }}
{{- end }}
{{- end }}

{{/*
ClickHouse image
*/}}
{{- define "edge-logs.clickhouse.image" -}}
{{- $registry := .Values.clickhouse.image.registry | default .Values.global.imageRegistry -}}
{{- if $registry }}
{{- printf "%s/%s:%s" $registry .Values.clickhouse.image.repository .Values.clickhouse.image.tag }}
{{- else }}
{{- printf "%s:%s" .Values.clickhouse.image.repository .Values.clickhouse.image.tag }}
{{- end }}
{{- end }}

{{/*
ClickHouse service name
*/}}
{{- define "edge-logs.clickhouse.serviceName" -}}
{{- printf "%s-clickhouse" (include "edge-logs.fullname" .) }}
{{- end }}

{{/*
APIServer service name
*/}}
{{- define "edge-logs.apiserver.serviceName" -}}
{{- printf "%s-apiserver" (include "edge-logs.fullname" .) }}
{{- end }}

{{/*
Frontend service name
*/}}
{{- define "edge-logs.frontend.serviceName" -}}
{{- printf "%s-frontend" (include "edge-logs.fullname" .) }}
{{- end }}

{{/*
OTEL Collector image
*/}}
{{- define "edge-logs.otelCollector.image" -}}
{{- $registry := .Values.otelCollector.image.registry | default .Values.global.imageRegistry -}}
{{- if $registry }}
{{- printf "%s/%s:%s" $registry .Values.otelCollector.image.repository .Values.otelCollector.image.tag }}
{{- else }}
{{- printf "%s:%s" .Values.otelCollector.image.repository .Values.otelCollector.image.tag }}
{{- end }}
{{- end }}

{{/*
iLogtail image
*/}}
{{- define "edge-logs.ilogtail.image" -}}
{{- $registry := .Values.ilogtail.image.registry | default .Values.global.imageRegistry -}}
{{- if $registry }}
{{- printf "%s/%s:%s" $registry .Values.ilogtail.image.repository .Values.ilogtail.image.tag }}
{{- else }}
{{- printf "%s:%s" .Values.ilogtail.image.repository .Values.ilogtail.image.tag }}
{{- end }}
{{- end }}

{{/*
Namespace
*/}}
{{- define "edge-logs.namespace" -}}
{{- default .Release.Namespace .Values.namespaceOverride }}
{{- end }}