import { AuthorizeRequest, AuthorizeResponse, CredentialServiceOptions, DelegateRequest, DelegateResponse, FetchLike, IssueRequest, IssueResponse, VerifyRequest, VerifyResponse } from './types.js';
export declare function buildUrl(baseUrl: string | undefined, path: string): string;
export declare class CredentialServiceClient {
    readonly issuerUrl?: string;
    readonly verifierUrl?: string;
    readonly gatewayUrl?: string;
    readonly fetchImpl: FetchLike;
    constructor({ issuerUrl, verifierUrl, gatewayUrl, fetchImpl }?: CredentialServiceOptions);
    issueVC(request: IssueRequest): Promise<IssueResponse>;
    issueSDJWT(request: IssueRequest): Promise<IssueResponse>;
    verify(credentialOrRequest: string | VerifyRequest): Promise<VerifyResponse>;
    authorize(request: AuthorizeRequest): Promise<AuthorizeResponse>;
    delegateCredential(request: DelegateRequest): Promise<DelegateResponse>;
    private issue;
    private post;
    private mapError;
}
export default CredentialServiceClient;
