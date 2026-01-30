{{/*
Expand the name of the chart.
*/}}
{{- define "uno.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this.
*/}}
{{- define "uno.fullname" -}}
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
{{- define "uno.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "uno.labels" -}}
helm.sh/chart: {{ include "uno.chart" . }}
{{ include "uno.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "uno.selectorLabels" -}}
app.kubernetes.io/name: {{ include "uno.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "uno.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "uno.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the secret to use
*/}}
{{- define "uno.secretName" -}}
{{- printf "%s-secret" (include "uno.fullname" .) }}
{{- end }}

{{/*
Restate worker fullname and selector labels
*/}}
{{- define "uno.restateWorkerFullname" -}}
{{- printf "%s-restate-worker" (include "uno.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "uno.restateWorkerSelectorLabels" -}}
{{ include "uno.selectorLabels" . }}
app.kubernetes.io/component: restate-worker
{{- end }}

{{/*
Temporal worker fullname and selector labels
*/}}
{{- define "uno.temporalWorkerFullname" -}}
{{- printf "%s-temporal-worker" (include "uno.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "uno.temporalWorkerSelectorLabels" -}}
{{ include "uno.selectorLabels" . }}
app.kubernetes.io/component: temporal-worker
{{- end }}


