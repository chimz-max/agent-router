# ai-gateway-helm

![Version: v0.0.0-latest](https://img.shields.io/badge/Version-v0.0.0--latest-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: latest](https://img.shields.io/badge/AppVersion-latest-informational?style=flat-square)

The Helm chart for Envoy AI Gateway

**Homepage:** <https://aigateway.envoyproxy.io/>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| envoy-ai-gateway-maintainers |  | <https://github.com/envoyproxy/ai-gateway/blob/main/CODEOWNERS> |

## Source Code

* <https://github.com/envoyproxy/ai-gateway>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| controller.affinity | object | `{}` |  |
| controller.extraEnvVars | list | `[]` |  |
| controller.fullnameOverride | string | `"ai-gateway-controller"` |  |
| controller.hostNetwork | bool | `false` |  |
| controller.image.repository | string | `"docker.io/envoyproxy/ai-gateway-controller"` |  |
| controller.image.tag | string | `""` |  |
| controller.imagePullPolicy | string | `"IfNotPresent"` |  |
| controller.imagePullSecrets | list | `[]` |  |
| controller.leaderElection.enabled | bool | `true` |  |
| controller.logLevel | string | `"info"` |  |
| controller.logRequestHeaderAttributes | string | `nil` |  |
| controller.mcp.sessionEncryption.fallback.iterations | int | `100000` |  |
| controller.mcp.sessionEncryption.fallback.seed | string | `""` |  |
| controller.mcp.sessionEncryption.iterations | int | `100000` |  |
| controller.mcp.sessionEncryption.seed | string | `"default-insecure-seed"` |  |
| controller.metricsRequestHeaderAttributes | string | `""` |  |
| controller.mutatingWebhook.caBundleName | string | `"ca.crt"` |  |
| controller.mutatingWebhook.certManager.certificateName | string | `"self-signed-cert-for-mutating-webhook"` |  |
| controller.mutatingWebhook.certManager.enable | bool | `false` |  |
| controller.mutatingWebhook.certManager.issuerName | string | `"self-signed-issuer-for-mutating-webhook"` |  |
| controller.mutatingWebhook.objectSelector.matchLabels."app.kubernetes.io/managed-by" | string | `"envoy-gateway"` |  |
| controller.mutatingWebhook.port | int | `9443` |  |
| controller.mutatingWebhook.tlsCertName | string | `"tls.crt"` |  |
| controller.mutatingWebhook.tlsCertSecretName | string | `"self-signed-cert-for-mutating-webhook"` |  |
| controller.mutatingWebhook.tlsKeyName | string | `"tls.key"` |  |
| controller.nameOverride | string | `""` |  |
| controller.nodeSelector | object | `{}` |  |
| controller.podAnnotations | object | `{}` |  |
| controller.podEnv | object | `{}` |  |
| controller.podSecurityContext | object | `{}` |  |
| controller.quotaRateLimitFailureModeDeny | bool | `false` |  |
| controller.quotaRateLimitServiceAddr | string | `"envoy-ai-gateway-ratelimit.envoy-gateway-system"` |  |
| controller.quotaRateLimitTimeout | int | `5` |  |
| controller.replicaCount | int | `1` |  |
| controller.requestHeaderAttributes | string | `""` |  |
| controller.resources | object | `{}` |  |
| controller.securityContext | object | `{}` |  |
| controller.service.ports[0].appProtocol | string | `"http"` |  |
| controller.service.ports[0].name | string | `"mutating-webhook"` |  |
| controller.service.ports[0].port | int | `9443` |  |
| controller.service.ports[0].protocol | string | `"TCP"` |  |
| controller.service.ports[0].targetPort | int | `9443` |  |
| controller.service.ports[1].appProtocol | string | `"grpc"` |  |
| controller.service.ports[1].name | string | `"grpc"` |  |
| controller.service.ports[1].port | int | `1063` |  |
| controller.service.ports[1].protocol | string | `"TCP"` |  |
| controller.service.ports[1].targetPort | int | `1063` |  |
| controller.service.ports[2].appProtocol | string | `"http"` |  |
| controller.service.ports[2].name | string | `"http-metrics"` |  |
| controller.service.ports[2].port | int | `8080` |  |
| controller.service.ports[2].protocol | string | `"TCP"` |  |
| controller.service.ports[2].targetPort | int | `8080` |  |
| controller.service.ports[3].appProtocol | string | `"grpc"` |  |
| controller.service.ports[3].name | string | `"ratelimit-xds"` |  |
| controller.service.ports[3].port | int | `18002` |  |
| controller.service.ports[3].protocol | string | `"TCP"` |  |
| controller.service.ports[3].targetPort | int | `18002` |  |
| controller.service.type | string | `"ClusterIP"` |  |
| controller.serviceAccount.annotations | object | `{}` |  |
| controller.serviceAccount.create | bool | `true` |  |
| controller.serviceAccount.name | string | `""` |  |
| controller.spanRequestHeaderAttributes | string | `nil` |  |
| controller.tolerations | list | `[]` |  |
| controller.volumes | list | `[]` |  |
| controller.watch.cacheSyncTimeout | string | `"2m"` |  |
| controller.watch.namespaces | list | `[]` |  |
| endpointConfig.anthropic | string | `"/anthropic"` |  |
| endpointConfig.cohere | string | `"/cohere"` |  |
| endpointConfig.openai | string | `""` |  |
| endpointConfig.rootPrefix | string | `"/"` |  |
| envoyGateway.namespace | string | `"envoy-gateway-system"` |  |
| extProc.enableRedaction | bool | `false` |  |
| extProc.extraEnvVars | list | `[]` |  |
| extProc.image.repository | string | `"docker.io/envoyproxy/ai-gateway-extproc"` |  |
| extProc.image.tag | string | `""` |  |
| extProc.imagePullPolicy | string | `"IfNotPresent"` |  |
| extProc.imagePullSecrets | list | `[]` |  |
| extProc.logLevel | string | `"info"` |  |
| extProc.traceEmbeddingVectors | bool | `true` | Capture valid embedding vectors in traces when validateEmbeddingVectors is false. Set false to record only model and token usage and avoid vector decoding for tracing. This option has no effect when validation is true; full decoding and existing tracing remain. Maps to controller --extProcTraceEmbeddingVectors and sidecar -traceEmbeddingVectors. |
| extProc.validateEmbeddingVectors | bool | `true` | Validate vectors in OpenAI/Azure OpenAI embedding responses (upstream behavior). Full decoding has substantially higher memory use: a 100 MiB response benchmark allocated about 2.66 GiB with validation, versus 256 MiB without it (not peak RSS). Set false for metadata-only passthrough: JSON syntax and model/usage are still validated; trace vectors are best-effort. Tracing valid vectors still costs memory. Maps to controller --extProcValidateEmbeddingVectors and sidecar -validateEmbeddingVectors. |
| global.imagePullSecrets | list | `[]` | references to pre-created imagePullSecrets in the target namespace Precedence: component-specific values win. If `controller.imagePullSecrets` or `extProc.imagePullSecrets` are set (non-empty), those are used; otherwise this `global.imagePullSecrets` value is used as a fallback (values are not merged). Example: imagePullSecrets:   - name: regcred |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.14.2](https://github.com/norwoodj/helm-docs/releases/v1.14.2)
