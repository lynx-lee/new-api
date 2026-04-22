{{/*
Expand the name of the chart.
*/}}
{{- define "ai-bridge.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "ai-bridge.fullname" -}}
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
{{- define "ai-bridge.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "ai-bridge.labels" -}}
helm.sh/chart: {{ include "ai-bridge.chart" . }}
{{ include "ai-bridge.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "ai-bridge.selectorLabels" -}}
app.kubernetes.io/name: {{ include "ai-bridge.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the DSN connection string
Supports: custom DSN, internal PostgreSQL (bitnami), external PostgreSQL Cluster
*/}}
{{- define "ai-bridge.database.dsn" -}}
{{- if .Values.database.dsn -}}
{{- .Values.database.dsn -}}
{{- else if .Values.database.external.enabled -}}
{{- printf "%s:%s@tcp(%s:%v)/%s?charset=utf8mb4&parseTime=True&loc=Local" .Values.database.external.user .Values.database.external.password .Values.database.external.host .Values.database.external.port .Values.database.external.database -}}
{{- else if eq .Values.database.type "postgresql" -}}
{{- printf "%s:%s@%s-postgresql(%v)/%s?sslmode=disable" .Values.database.user (include "ai-bridge.dbPassword" .) (include "ai-bridge.fullname" .) .Values.database.port .Values.database.database -}}
{{- else if eq .Values.database.type "mysql" -}}
{{- printf "%s:%s@tcp(%s-mysql:%v)/%s?charset=utf8mb4&parseTime=True&loc=Local" .Values.database.user (include "ai-bridge.dbPassword" .) (include "ai-bridge.fullname" .) .Values.database.port .Values.database.database -}}
{{- end -}}
{{- end }}

{{/*
Get database password from secret or values
*/}}
{{- define "ai-bridge.dbPassword" -}}
{{- if .Values.database.password -}}
{{- .Values.database.password -}}
{{- else -}}
{{- .Values.postgresql.auth.password -}}
{{- end -}}
{{- end }}

{{/*
Create the Redis connection string
Supports: external Redis, standalone (bitnami), cluster mode
*/}}
{{- define "ai-bridge.redis.connectionString" -}}
{{- if .Values.redis.external.enabled -}}
{{/* External Redis (standalone or managed service) */}}
{{- printf "redis://:%s@%s:%v/%d" .Values.redis.external.password .Values.redis.external.host .Values.redis.external.port .Values.redis.external.db -}}
{{- else if eq .Values.redis.architecture "cluster" -}}
{{/* Redis Cluster mode */}}
{{- $nodes := .Values.redis.cluster.nodes -}}
{{- if $nodes -}}
{{- range $idx, $node := $nodes -}}
{{- if eq $idx 0 -}}{{ printf "redis://:%s@%s/0" .Values.redis.cluster.password $node }}{{ end -}}
{{- if ne $idx 0 -}}{{ printf "&addr=%s" $node }}{{ end -}}
{{- end -}}
{{- else -}}
{{- /* Fallback to redis master for cluster without explicit nodes */}}
{{- printf "redis://:%s@%s-redis-master:%v/0" .Values.redis.auth.password (include "ai-bridge.fullname" .) .Values.redis.service.port -}}
{{- end -}}
{{- else -}}
{{/* Standalone mode (default, bitnami redis sub-chart) */}}
{{- printf "redis://:%s@%s-redis-master:%v/0" .Values.redis.auth.password (include "ai-bridge.fullname" .) .Values.redis.service.port -}}
{{- end -}}
{{- end }}
