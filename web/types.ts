export interface Parameter {
  data_type: "string" | "number" | "boolean"
  value_type: "dynamic" | "constant"
  description: string
  identifier: string
  required: boolean
  value?: string
}

export interface Header {
  type: "constant" | "dynamic"
  name: string
  value: string
}

export interface Endpoint {
  capability: "tool" | "prompt"
  mode: "client" | "server"
  name: string
  path: string
  method: "GET" | "POST" | "PUT" | "DELETE" | "PATCH"
  description: string
  wait_response: boolean
  response_timeout: string
  body_params?: Parameter[]
  query_parameters?: Parameter[]
  path_parameters?: Parameter[]
}

export interface ApiService {
  base_url: string
  default_headers: Header[]
  endpoints: Endpoint[]
}
