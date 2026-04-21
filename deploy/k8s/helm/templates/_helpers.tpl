{{/*
Expand the name of the chart.
*/}}
{{- define "new-api.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "new-api.fullname" -}}
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
{{- define "new-api.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "new-api.labels" -}}
helm.sh/chart: {{ include "new-api.chart" . }}
{{ include "new-api.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "new-api.selectorLabels" -}}
app.kubernetes.io/name: {{ include "new-api.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the DSN connection string
*/}}
{{- define "new-api.database.dsn" -}}
{{- if .Values.database.dsn -}}
{{- .Values.database.dsn -}}
{{- else if eq .Values.database.type "postgresql" -}}
{{- printf "%s:%s@%s-new-api(%v)/%s?sslmode=disable" .Values.database.user .Values.database.password (include "new-api.fullname" .) .Values.database.port .Values.database.database -}}
{{- else if eq .Values.database.type "mysql" -}}
{{- printf "%s:%s@tcp(%s-new-api:%v)/%s" .Values.database.user .Values.database.password (include "new-api.fullname" .) .Values.database.port .Values.database.database -}}
{{- end -}}
{{- end }}

{{/*
Create the Redis connection string
*/}}
{{- define "new-api.redis.connectionString" -}}
{{- if .Values.redis.external.enabled -}}
{{- printf "redis://:%s@%s:%v/%d" .Values.redis.external.password .Values.redis.external.host .Values.redis.external.port .Values.redis.external.db -}}
{{- else -}}
{{- printf "redis://:%s@%s-redis-master:%v/0" .Values.redis.auth.password (include "new-api.fullname" .) .Values.redis.service.port -}}
{{- end -}}
{{- end }}
