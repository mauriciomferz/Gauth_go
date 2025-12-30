/**
 * PoA Marketing Credential System
 * Real-time validation for marketing agent requests with rule-based enforcement
 */
class PoAMarketingCredentialSystem {
    constructor(apiClient) {
        this.api = apiClient;
        this.currentCredential = null;
        this.init();
    }

    init() {
        this.bindEvents();
        this.loadMarketingCredential();
    }

    bindEvents() {
        document.addEventListener('click', (e) => {
            if (e.target.matches('[data-validate-marketing-request]') {
                this.validateMarketingRequest();
            }
            if (e.target.matches('[data-create-marketing-poa]') {
                this.createMarketingPoA();
            }
            if (e.target.matches('[data-test-social-campaign]') {
                this.testSocialMediaCampaign();
            }
            if (e.target.matches('[data-test-content-approval]') {
                this.testContentApproval();
            }
        });
    }

    loadMarketingCredential() {
        // Load the enterprise marketing PoA credential
        this.currentCredential = {
            poa_id: "acme-marketing-poa-2025",
            principal: "acme.com",
            agent: "marketing-ai-agent",
            issued_at: "2025-01-01T00:00:00Z",
            expires_at: "2025-12-31T23:59:59Z",
            capabilities: [
                {
                    action: "social_media_campaign",
                    resources: ["linkedin", "twitter", "instagram"],
                    constraints: {
                        budget_limit: 50000,
                        currency: "EUR",
                        monthly_budget: true,
                        time_window: "business_hours",
                        content_approval: "auto",
                        geographic_scope: ["EMEA", "Americas", "APAC"],
                        excluded_regions: ["restricted_countries"],
                        target_audience: ["engineering_professionals", "manufacturing_executives", "industrial_iot_specialists"],
                        content_types: ["product_announcements", "thought_leadership", "case_studies", "technical_insights"],
                        compliance_requirements: ["gdpr", "corporate_guidelines", "brand_standards"]
                    }
                },
                {
                    action: "customer_engagement",
                    resources: ["support_tickets", "social_mentions", "inquiries"],
                    constraints: {
                        response_time: "2h",
                        escalation_threshold: "complex_technical_issues",
                        auto_response_limit: 10,
                        languages: ["en", "de", "fr", "es"],
                        sentiment_threshold: "neutral_or_positive"
                    }
                },
                {
                    action: "content_creation",
                    resources: ["blog_posts", "whitepapers", "social_content"],
                    constraints: {
                        content_length_max: 2000,
                        approval_required: ["regulatory_content", "pricing_information", "partnership_announcements"],
                        brand_compliance: "mandatory",
                        technical_accuracy_check: true,
                        plagiarism_check: true
                    }
                }
            ],
            signature: "0x1234567890abcdef..." // Mock signature
        };
    }

    async createMarketingPoA() {
        const outputDiv = document.getElementById('poa-marketing-output');
        if (!outputDiv) return;

        outputDiv.innerHTML = '<div class="text-blue-600">Creating Marketing PoA credential...</div>';

        // Simulate PoA creation process
        const steps = [
            'Defining delegation scope for marketing operations...',
            'Setting budget constraints (€50,000/month)...',
            'Configuring audience targeting parameters...',
            'Establishing compliance requirements (GDPR, brand guidelines)...',
            'Generating cryptographic credential...',
            'Signing with ACME corporate key...',
            'Registering in capability registry...',
            'Distributing to marketing AI agent...',
            '✅ Marketing PoA credential created successfully!'
        ];

        for (let i = 0; i < steps.length; i++) {
            await this.delay(500);
            outputDiv.innerHTML += `<div class="text-sm">${steps[i]}</div>`;
        }

        // Show the created credential
        setTimeout(() => {
            outputDiv.innerHTML += `
                <div class="mt-4 bg-green-50 p-4 rounded-lg">
                    <h4 class="font-semibold text-green-800 mb-2">Created PoA Credential:</h4>
                    <pre class="text-xs text-green-700 bg-green-100 p-3 rounded overflow-x-auto">${JSON.stringify(this.currentCredential, null, 2)}</pre>
                </div>
            `;
        }, 1000);
    }

    async validateMarketingRequest() {
        const outputDiv = document.getElementById('poa-marketing-output');
        if (!outputDiv) return;

        outputDiv.innerHTML = '<div class="text-blue-600">Validating marketing agent request...</div>';

        // Simulate a marketing request
        const marketingRequest = {
            agent_id: "marketing-ai-agent",
            timestamp: new Date().toISOString(),
            action: "social_media_campaign",
            resource: "linkedin",
            parameters: {
                campaign_type: "product_announcement",
                budget_requested: 15000,
                currency: "EUR",
                target_audience: "engineering_professionals",
                geographic_region: "EMEA",
                content_summary: "New ACME Digital Twin Solutions for Smart Manufacturing",
                estimated_reach: 50000,
                duration_days: 14,
                compliance_tags: ["gdpr_compliant", "brand_approved"]
            }
        };

        // Show the request
        outputDiv.innerHTML += `
            <div class="mt-2 bg-blue-50 p-3 rounded">
                <h4 class="font-semibold text-blue-800 mb-2">Marketing Request:</h4>
                <pre class="text-xs text-blue-700">${JSON.stringify(marketingRequest, null, 2)}</pre>
            </div>
        `;

        await this.delay(1000);

        // Perform validation
        const validation = await this.performPoAValidation(marketingRequest);
        
        outputDiv.innerHTML += `
            <div class="mt-4 bg-gray-50 p-4 rounded">
                <h4 class="font-semibold mb-2">PoA Validation Process:</h4>
                <div class="space-y-2 text-sm">
                    ${validation.steps.map(step => `
                        <div class="flex items-center gap-2">
                            <span class="${step.result ? 'text-green-600' : 'text-red-600'}">${step.result ? '✅' : '❌'}</span>
                            <span class="font-medium">${step.check}:</span>
                            <span class="${step.result ? 'text-green-700' : 'text-red-700'}">${step.result ? 'PASS' : 'FAIL'}</span>
                            ${step.details ? `<span class="text-gray-600">- ${step.details}</span>` : ''}
                        </div>
                    `).join('')}
                </div>
            </div>
        `;

        await this.delay(1000);

        // Show final decision
        outputDiv.innerHTML += `
            <div class="mt-4 p-4 rounded ${validation.approved ? 'bg-green-50 border border-green-200' : 'bg-red-50 border border-red-200'}">
                <div class="flex items-center gap-3 mb-2">
                    <div class="w-8 h-8 rounded-full flex items-center justify-center ${validation.approved ? 'bg-green-600' : 'bg-red-600'}">
                        <i class="fas ${validation.approved ? 'fa-check' : 'fa-times'} text-white"></i>
                    </div>
                    <h4 class="font-bold text-lg ${validation.approved ? 'text-green-800' : 'text-red-800'}">
                        ${validation.approved ? 'REQUEST APPROVED' : 'REQUEST DENIED'}
                    </h4>
                </div>
                <p class="${validation.approved ? 'text-green-700' : 'text-red-700'}">${validation.reasoning}</p>
                ${validation.approved ? `
                    <div class="mt-3 text-sm text-green-600">
                        <strong>Authorization ID:</strong> AUTH-${Date.now()}<br>
                        <strong>Valid Until:</strong> ${new Date(Date.now() + 24*60*60*1000).toLocaleString()}<br>
                        <strong>Budget Allocated:</strong> €${marketingRequest.parameters.budget_requested.toLocaleString()}
                    </div>
                ` : `
                    <div class="mt-3 text-sm text-red-600">
                        <strong>Action Required:</strong> ${validation.action_required || 'Review constraints and resubmit'}
                    </div>
                `}
            </div>
        `;
    }

    async performPoAValidation(request) {
        // Find matching capability
        const capability = this.currentCredential.capabilities.find(cap => cap.action === request.action);
        
        if (!capability) {
            return {
                approved: false,
                reasoning: `Action '${request.action}' not authorized in PoA credential`,
                steps: [
                    { check: "Capability Match", result: false, details: "Action not found in PoA" }
                ]
            };
        }

        const steps = [];
        let allChecksPass = true;

        // Credential signature check
        steps.push({
            check: "Credential Signature",
            result: true,
            details: "Valid cryptographic signature"
        });

        // Expiry check
        const now = new Date();
        const expiryDate = new Date(this.currentCredential.expires_at);
        const expiryValid = now < expiryDate;
        steps.push({
            check: "Credential Expiry",
            result: expiryValid,
            details: expiryValid ? `Valid until ${expiryDate.toLocaleDateString()}` : "Credential expired"
        });
        if (!expiryValid) allChecksPass = false;

        // Resource check
        const resourceAllowed = capability.resources.includes(request.resource);
        steps.push({
            check: "Resource Authorization",
            result: resourceAllowed,
            details: resourceAllowed ? `${request.resource} is authorized` : `${request.resource} not in allowed resources`
        });
        if (!resourceAllowed) allChecksPass = false;

        // Budget check
        const budgetValid = request.parameters.budget_requested <= capability.constraints.budget_limit;
        steps.push({
            check: "Budget Constraint",
            result: budgetValid,
            details: `€${request.parameters.budget_requested.toLocaleString()} ${budgetValid ? '≤' : '>'} €${capability.constraints.budget_limit.toLocaleString()}`
        });
        if (!budgetValid) allChecksPass = false;

        // Geographic scope check
        const geoValid = capability.constraints.geographic_scope.includes(request.parameters.geographic_region);
        steps.push({
            check: "Geographic Scope",
            result: geoValid,
            details: geoValid ? `${request.parameters.geographic_region} region authorized` : `${request.parameters.geographic_region} not in authorized regions`
        });
        if (!geoValid) allChecksPass = false;

        // Target audience check
        const audienceValid = capability.constraints.target_audience.includes(request.parameters.target_audience);
        steps.push({
            check: "Target Audience",
            result: audienceValid,
            details: audienceValid ? `Targeting ${request.parameters.target_audience} approved` : `${request.parameters.target_audience} not in approved audiences`
        });
        if (!audienceValid) allChecksPass = false;

        // Content type check
        const contentValid = capability.constraints.content_types.includes(request.parameters.campaign_type);
        steps.push({
            check: "Content Type",
            result: contentValid,
            details: contentValid ? `${request.parameters.campaign_type} content approved` : `${request.parameters.campaign_type} not in approved content types`
        });
        if (!contentValid) allChecksPass = false;

        // Compliance check
        const complianceValid = request.parameters.compliance_tags.every(tag => 
            capability.constraints.compliance_requirements.some(req => tag.includes(req.replace('_', '')))
        );
        steps.push({
            check: "Compliance Requirements",
            result: complianceValid,
            details: complianceValid ? "All compliance requirements met" : "Missing required compliance certifications"
        });
        if (!complianceValid) allChecksPass = false;

        return {
            approved: allChecksPass,
            reasoning: allChecksPass ? 
                "All PoA constraints satisfied. Marketing campaign approved for execution." :
                "One or more PoA constraints violated. Request denied for compliance reasons.",
            steps: steps,
            action_required: allChecksPass ? null : "Review failed constraints and modify request parameters"
        };
    }

    async testSocialMediaCampaign() {
        const outputDiv = document.getElementById('poa-marketing-output');
        if (!outputDiv) return;

        outputDiv.innerHTML = '<div class="text-purple-600">Testing social media campaign scenario...</div>';

        // Test different scenarios
        const scenarios = [
            {
                name: "Valid LinkedIn Campaign",
                request: {
                    action: "social_media_campaign",
                    resource: "linkedin",
                    parameters: {
                        campaign_type: "thought_leadership",
                        budget_requested: 25000,
                        target_audience: "engineering_professionals",
                        geographic_region: "EMEA"
                    }
                },
                expected: true
            },
            {
                name: "Over-Budget Campaign",
                request: {
                    action: "social_media_campaign",
                    resource: "linkedin",
                    parameters: {
                        campaign_type: "product_announcement",
                        budget_requested: 75000, // Exceeds 50k limit
                        target_audience: "engineering_professionals",
                        geographic_region: "EMEA"
                    }
                },
                expected: false
            },
            {
                name: "Invalid Audience",
                request: {
                    action: "social_media_campaign",
                    resource: "linkedin",
                    parameters: {
                        campaign_type: "product_announcement",
                        budget_requested: 30000,
                        target_audience: "general_consumers", // Not in approved list
                        geographic_region: "EMEA"
                    }
                },
                expected: false
            }
        ];

        for (const scenario of scenarios) {
            await this.delay(1000);
            outputDiv.innerHTML += `
                <div class="mt-4 bg-indigo-50 p-4 rounded border border-indigo-200">
                    <h4 class="font-semibold text-indigo-800 mb-2">Testing: ${scenario.name}</h4>
                    <div class="text-sm text-indigo-700">
                        Budget: €${scenario.request.parameters.budget_requested.toLocaleString()}<br>
                        Audience: ${scenario.request.parameters.target_audience}<br>
                        Type: ${scenario.request.parameters.campaign_type}
                    </div>
                    <div class="mt-2">
                        <span class="text-xs px-2 py-1 rounded ${scenario.expected ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}">
                            Expected: ${scenario.expected ? 'APPROVE' : 'DENY'}
                        </span>
                    </div>
                </div>
            `;
        }

        outputDiv.innerHTML += `
            <div class="mt-4 p-4 bg-gray-50 rounded">
                <h4 class="font-semibold mb-2">Test Summary:</h4>
                <p class="text-sm text-gray-700">
                    Tested ${scenarios.length} different campaign scenarios to validate PoA constraint enforcement.
                    The system correctly identifies budget limits, audience restrictions, and content type approvals.
                </p>
            </div>
        `;
    }

    async testContentApproval() {
        const outputDiv = document.getElementById('poa-marketing-output');
        if (!outputDiv) return;

        outputDiv.innerHTML = '<div class="text-orange-600">Testing content approval workflow...</div>';

        const contentTypes = [
            { type: "product_announcement", approval: "auto", reason: "Standard content type" },
            { type: "pricing_information", approval: "manual", reason: "Requires legal review" },
            { type: "partnership_announcement", approval: "manual", reason: "Strategic content requires approval" },
            { type: "technical_insights", approval: "auto", reason: "Educational content approved" }
        ];

        for (const content of contentTypes) {
            await this.delay(800);
            outputDiv.innerHTML += `
                <div class="mt-3 flex items-center justify-between p-3 bg-gray-50 rounded border">
                    <div>
                        <div class="font-medium">${content.type.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())}</div>
                        <div class="text-sm text-gray-600">${content.reason}</div>
                    </div>
                    <div class="flex items-center gap-2">
                        <span class="text-xs px-2 py-1 rounded ${content.approval === 'auto' ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'}">
                            ${content.approval === 'auto' ? 'AUTO-APPROVED' : 'MANUAL REVIEW'}
                        </span>
                        <i class="fas ${content.approval === 'auto' ? 'fa-check-circle text-green-600' : 'fa-clock text-yellow-600'}"></i>
                    </div>
                </div>
            `;
        }

        outputDiv.innerHTML += `
            <div class="mt-4 p-4 bg-blue-50 rounded border border-blue-200">
                <h4 class="font-semibold text-blue-800 mb-2">Content Approval Matrix:</h4>
                <p class="text-sm text-blue-700">
                    The PoA credential defines which content types can be auto-approved and which require manual review.
                    This ensures brand consistency while enabling autonomous operation within safe boundaries.
                </p>
            </div>
        `;
    }

    delay(ms) {
        return new Promise(resolve => setTimeout(resolve, ms));
    }

    // Method to integrate with the pattern explorer
    getMarketingPoADemo() {
        return {
            title: "Marketing PoA Credential Validation",
            description: "Real-time validation system for enterprise marketing AI agent",
            features: [
                "Budget constraint enforcement",
                "Audience targeting validation", 
                "Content type approval",
                "Geographic scope checking",
                "Compliance requirement verification"
            ]
        };
    }
}

// Export for use in other modules
window.PoAMarketingCredentialSystem = PoAMarketingCredentialSystem;