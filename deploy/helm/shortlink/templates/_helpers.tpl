{{- define "shortlink.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- if .Values.serviceAccount.name -}}
{{ .Values.serviceAccount.name }}
{{- else -}}
shortlink
{{- end -}}
{{- else -}}
default
{{- end -}}
{{- end -}}


