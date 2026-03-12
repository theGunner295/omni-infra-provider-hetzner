{{/*
Expand the name of the chart.
*/}}
{{- define "omni-infra-provider-hetzner.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains the chart name it will be used as a full name.
*/}}
{{- define "omni-infra-provider-hetzner.fullname" -}}
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
Create chart label.
*/}}
{{- define "omni-infra-provider-hetzner.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "omni-infra-provider-hetzner.labels" -}}
helm.sh/chart: {{ include "omni-infra-provider-hetzner.chart" . }}
{{ include "omni-infra-provider-hetzner.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels.
*/}}
{{- define "omni-infra-provider-hetzner.selectorLabels" -}}
app.kubernetes.io/name: {{ include "omni-infra-provider-hetzner.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use.
*/}}
{{- define "omni-infra-provider-hetzner.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "omni-infra-provider-hetzner.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Return the name of the Secret containing sensitive configuration.
Uses existingSecret if provided, otherwise falls back to the generated fullname.
*/}}
{{- define "omni-infra-provider-hetzner.secretName" -}}
{{- if .Values.existingSecret }}
{{- .Values.existingSecret }}
{{- else }}
{{- include "omni-infra-provider-hetzner.fullname" . }}
{{- end }}
{{- end }}
