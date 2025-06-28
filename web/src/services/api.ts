import axios from "axios";

const API_BASE_URL = "/api";

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    "Content-Type": "application/json",
    Authorization: "Bearer 123456",
  },
});

// MCP代理专用的axios实例（不使用/api前缀）
const mcpApi = axios.create({
  baseURL: "/",
  headers: {
    "Content-Type": "application/json",
    Authorization: "Bearer 123456",
  },
});

// Types
export interface WorkspaceInfo {
  id: string;
  status: string;
  service_count: number;
  session_count: number;
  services?: ServiceInfo[];
}

export interface ServiceInfo {
  name: string;
  status: string;
  config: {
    command?: string;
    url?: string;
    workspace: string;
  };
  port: number;
  last_error?: string;
  failure_reason?: string;
  deployed_at: string;
  last_started_at?: string;
  last_stopped_at?: string;
  retry_count: number;
  retry_max: number;
  urls: {
    base_url?: string;
    sse_url?: string;
    message_url?: string;
  };
}

export interface SessionInfo {
  id: string;
  workspace_id: string;
  status: string;
  created_at: string;
  last_receive_time: string;
  is_ready: boolean;
}

export interface MCPServerConfig {
  command?: string;
  url?: string;
  args?: string[];
  env?: Record<string, string>;
  workspace?: string;
}

// Workspace API
export const workspaceApi = {
  getAll: () => api.get<WorkspaceInfo[]>("/workspaces"),
  create: (id: string) => api.post("/workspaces", { id }),
  delete: (id: string) => api.delete(`/workspaces/${id}`),
  getServices: (id: string) =>
    api.get<ServiceInfo[]>(`/workspaces/${id}/services`),
};

// Service API
export const serviceApi = {
  deploy: (workspaceId: string, services: Record<string, MCPServerConfig>) =>
    api.post(`/workspaces/${workspaceId}/services`, { mcp_servers: services }),
  update: (workspaceId: string, name: string, config: MCPServerConfig) =>
    api.put(`/workspaces/${workspaceId}/services/${name}`, config),
  restart: (workspaceId: string, name: string) =>
    api.post(`/workspaces/${workspaceId}/services/${name}/restart`),
  stop: (workspaceId: string, name: string) =>
    api.post(`/workspaces/${workspaceId}/services/${name}/stop`),
  start: (workspaceId: string, name: string) =>
    api.post(`/workspaces/${workspaceId}/services/${name}/start`),
  delete: (workspaceId: string, name: string) =>
    api.delete(`/workspaces/${workspaceId}/services/${name}`),
  getLogs: (workspaceId: string, name: string) =>
    api.get(`/workspaces/${workspaceId}/services/${name}/logs`),
};

// Session API
export const sessionApi = {
  getByWorkspace: (workspaceId: string) =>
    api.get<SessionInfo[]>(`/workspaces/${workspaceId}/sessions`),
  create: (workspaceId: string) =>
    api.post<SessionInfo>(`/workspaces/${workspaceId}/sessions`),
  delete: (workspaceId: string, sessionId: string) =>
    api.delete(`/workspaces/${workspaceId}/sessions/${sessionId}`),
  getStatus: (sessionId: string) =>
    api.get<SessionInfo>(`/sessions/${sessionId}/status`),
};

// Debug API Types
export interface DebugRequest {
  message: string;
  method?: string;
}

export interface DebugResponse {
  success: boolean;
  response?: Record<string, any>;
  error?: string;
  service_info: ServiceInfo;
  request_log?: string;
  response_log?: string;
}

export interface ConnectionTestResult {
  service_name: string;
  workspace: string;
  status: string;
  healthy: boolean;
  urls: {
    base_url?: string;
    sse_url?: string;
    message_url?: string;
  };
  tests: Array<{
    name: string;
    success: boolean;
    details: string;
    error?: string;
  }>;
  overall_success: boolean;
  success_rate: string;
}

export interface LogEntry {
  timestamp: string;
  level: string;
  message: string;
}

export interface ServiceLogsResponse {
  service_name: string;
  logs: LogEntry[];
  total_lines: number;
}

// Debug API
export const debugApi = {
  getInfo: (workspaceId: string, serviceName: string) =>
    api.get(`/workspaces/${workspaceId}/services/${serviceName}/debug/info`),

  testService: (
    workspaceId: string,
    serviceName: string,
    request: DebugRequest,
  ) =>
    api.post<DebugResponse>(
      `/workspaces/${workspaceId}/services/${serviceName}/debug/test`,
      request,
    ),

  testConnection: (workspaceId: string, serviceName: string) =>
    api.get<ConnectionTestResult>(
      `/workspaces/${workspaceId}/services/${serviceName}/debug/connection`,
    ),

  getLogs: (
    workspaceId: string,
    serviceName: string,
    limit?: number,
    offset?: number,
  ) =>
    api.get<ServiceLogsResponse>(
      `/workspaces/${workspaceId}/services/${serviceName}/debug/logs`,
      {
        params: { limit, offset },
      },
    ),
};

// SSE Connection Helper
export class SSEConnection {
  private eventSource: EventSource | null = null;
  private url: string;
  private onMessage: (data: any) => void;
  private onError: (error: Event) => void;
  private onOpen: () => void;

  constructor(
    url: string,
    onMessage: (data: any) => void,
    onError: (error: Event) => void = () => {},
    onOpen: () => void = () => {},
  ) {
    this.url = url;
    this.onMessage = onMessage;
    this.onError = onError;
    this.onOpen = onOpen;
  }

  // 静态方法：为MCP服务创建SSE URL（通过代理）
  static createSSEUrl(serviceName: string, workspaceId: string): string {
    // EventSource 不支持自定义 headers，所以我们将workspace信息作为查询参数传递
    return `http://localhost:8080/${serviceName}/sse?workspaceId=${encodeURIComponent(workspaceId)}&api_key=123456`;
  }

  connect() {
    if (this.eventSource) {
      this.disconnect();
    }

    this.eventSource = new EventSource(this.url);

    this.eventSource.onopen = () => {
      console.log("SSE connection opened");
      this.onOpen();
    };

    this.eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        this.onMessage(data);
      } catch (error) {
        console.error("Failed to parse SSE message:", error);
        this.onMessage({ raw: event.data, error: "Failed to parse message" });
      }
    };

    this.eventSource.onerror = (error) => {
      console.error("SSE connection error:", error);
      this.onError(error);
    };
  }

  disconnect() {
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
  }

  isConnected(): boolean {
    return (
      this.eventSource !== null &&
      this.eventSource.readyState === EventSource.OPEN
    );
  }
}

// MCP Message Sending Helper
export class MCPMessageSender {
  private serviceName: string;
  private workspaceId: string;

  constructor(serviceName: string, workspaceId: string) {
    this.serviceName = serviceName;
    this.workspaceId = workspaceId;
  }

  async sendMessage(message: object): Promise<any> {
    const response = await mcpApi.post(`${this.serviceName}/message`, message, {
      headers: {
        "X-Workspace-Id": this.workspaceId,
      },
    });
    return response.data;
  }

  async sendJSONRPCMessage(
    method: string,
    params: any = {},
    id: number | string = Date.now(),
  ): Promise<any> {
    const message = {
      jsonrpc: "2.0",
      id,
      method,
      params,
    };

    return await this.sendMessage(message);
  }

  // 常用的MCP方法
  async ping(): Promise<any> {
    return this.sendJSONRPCMessage("ping");
  }

  async initialize(clientInfo: any): Promise<any> {
    return this.sendJSONRPCMessage("initialize", clientInfo);
  }

  async listTools(): Promise<any> {
    return this.sendJSONRPCMessage("tools/list");
  }

  async callTool(name: string, arguments_: any): Promise<any> {
    return this.sendJSONRPCMessage("tools/call", {
      name,
      arguments: arguments_,
    });
  }

  async listResources(): Promise<any> {
    return this.sendJSONRPCMessage("resources/list");
  }

  async readResource(uri: string): Promise<any> {
    return this.sendJSONRPCMessage("resources/read", { uri });
  }
}

// API Debug Types
export interface APIParameter {
  name: string;
  type: string;
  location: string; // path, query, body, header
  required: boolean;
  description?: string;
  example?: string;
}

export interface APIExample {
  name: string;
  description?: string;
  request?: Record<string, any>;
  response?: Record<string, any>;
}

export interface APIEndpoint {
  method: string;
  path: string;
  handler: string;
  middleware?: string[];
  parameters?: APIParameter[];
  description?: string;
  group?: string;
  tags?: string[];
  examples?: APIExample[];
}

export interface APIDiscoveryResponse {
  total_endpoints: number;
  groups: string[];
  endpoints: APIEndpoint[];
  generated_at: string;
  version: string;
}

export interface APITestRequest {
  method: string;
  path: string;
  headers?: Record<string, string>;
  query?: Record<string, string>;
  body?: Record<string, any>;
  content_type?: string;
}

export interface APITestResponse {
  success: boolean;
  status_code: number;
  response_time: number; // in nanoseconds
  response?: Record<string, any>;
  error?: string;
  request_headers?: Record<string, string>;
  request_body?: string;
  response_body?: string;
}

export interface APIGroupsResponse {
  groups: Record<string, {
    description: string;
    endpoints: string[];
  }>;
  total_groups: number;
}

// API Debug API
export const apiDebugApi = {
  // 获取所有API列表
  discoverAPIs: () => api.get<APIDiscoveryResponse>("/debug/apis"),
  
  // 获取API分组信息
  getAPIGroups: () => api.get<APIGroupsResponse>("/debug/apis/groups"),
  
  // 测试API端点
  testAPI: (request: APITestRequest) => 
    api.post<APITestResponse>("/debug/apis/test", request),
};

export default api;
