export interface IssueRequest {
  subject_did: string;
  ttl_seconds: number;
  claims?: Record<string, any>;
}

export interface DelegateRequest {
  parent_credential: string;
  delegate_did: string;
  scope: string[];
  ttl_seconds: number;
  claims?: Record<string, any>;
}

export interface GatewayAuthorizeRequest {
  credential?: string;
  credentials?: string[];
  expected_audience?: string;
  want_synthetic_jwt?: boolean;
}

export interface IssueResponse {
  credential: string;
  subject?: string;
  acting_on_behalf_of?: string;
  delegation_depth?: number;
  claims?: Record<string, any>;
  synthetic_jwt?: string;
}

export interface GatewayAuthorizeResponse {
  allowed: boolean;
  subject?: string;
  acting_on_behalf_of?: string;
  delegation_depth?: number;
  claims?: Record<string, any>;
  reason?: string;
  synthetic_jwt?: string;
}

export class Client {
  constructor(config: { baseUrl: string; fetchImpl?: typeof fetch });
  issueCredential(request: IssueRequest): Promise<IssueResponse>;
  delegateCredential(request: DelegateRequest): Promise<IssueResponse>;
  verify(token: string): Promise<any>;
  gatewayAuthorize(request: GatewayAuthorizeRequest): Promise<GatewayAuthorizeResponse>;
}
