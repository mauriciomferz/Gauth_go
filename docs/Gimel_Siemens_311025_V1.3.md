---
title: Gimel Siemens 311025 V1.3
category: guide
status: draft
lastUpdated: 2025-12-25
owners: [system]
---

G imel
The Open-Source Platform
for AI Control and Secure IDs

- Discussion paper, V1.3 T0 Mrs.
Constanze Pott and Martina Vollmer
Siemens AG
5. November 2025
- All rights reserved, confidential -

Do

no

ts

ha

re
–

fo
r

int

er

na

lp

ur

po

se

so

nly

CGTcoin
Status summary
•

Target: Gimel is the open-source platform for Agent Control and Secure IDs. Siemens and Gimel are discussing a collaboration, to deploy Gimel

•

Security beyond High Assurance: Gimel is enabling security beyond high assurance (LoA4), through DNA based biometrics (Gimel ID),

•

Architecture fit: Gimel is complementing the existing architecture of Siemens, like EntraID. While EntraID provides the protocol for granting

at the Agent Factory of Siemens to facilitate secure operations and AI governance. The parties have signed an NDA including IP protection
agreement, accordingly. Gimel Technologies has filed patents for its solutions.
Web3-based AI authorization (GAuth+) and continuous monitoring of both (DefconG, G-Agent).

system access, Gimel is orchestrating the power of attorney (PoA) for agents and humanoid robots with a separate authorization server. The
EntraID protocol can be uplifted towards more secure biometrics, i.e., with Gimel ID. Gimel can leverage on existing infrastructure providers of
Siemens, such as PKI providers and cloud service providers.

•

Compliance: Gimel is designed to meet relevant requirements such as GDPR, eIDAS 2.0, those of cyber-security regulators like the BSI, ENISA

and NIST, relevant industry foundations like IETF and OIDC as well as the key concepts like Self-Sovereign Identity (SSI), Law of Agency and Least
Privilege. The collaboration will drive from a beta-test towards full compliance. No other solution does that.

•
•

•

Enforcement: Gimel enforces PoA-compliance through three layers, rules, AI (G-Agent) and PoA-disclosure (towards relying party). In line with

other cybersecurity protocols (e.g., IDS/IPS, NGFWs, SOAR), Gimel combines deterministic, signature-like and non-deterministic, anomalyfocused procedures, while mitigating remaining risks through a legal mechanism (disclosure to relying party).
Use Cases: Digital agents and humanoid robots provide the opportunity to implement various new use cases such as the Autonomous
Marketing Agent (example). The use of AI in Marketing develops from RAG systems towards autonomously acting agents, accordingly. Gimel,
therefore, orchestrates agent compliance based on its PoA-credential, capturing the details of such functions while considering the leastprivilege principle. Such functional use cases become increasingly relevant to Siemens as the split between Reactive RAG versus Autonomous AI
develops from approx. 80:20 today to some 25:75 by 2028.

Proof of Concept: To facilitate a jump start and fast progress, Siemens can leverage on existing coding of the open-source protocol GAuth. In

this case, the GAuth workstream for setting up the authorization server being performed by Siemens with Gimel support. Other PoC-workstreams
include the G-Agent, GAuth+/Web3, Gimel ID / Lab and Gimel Wallet / Cloud.

- All rights reserved -

1

CGTcoin
Agenda

1

Why Gimel - Problem statement and Gimel solution

2

Architecture fit - Gimel complementing Siemens architecture

3

Example deep dive – Use case “Marketing Agent”

4

How to get started - Gimel PoC for Siemens and implementation planning

- All rights reserved -

2

CGTcoin
The problem: Current cost of cybercrime approaches
USD 10 trillion* growing due to AI and its challenges
The identity challenge

The agent-control challenge

1

•

Fake identities due to weak biometrics
(fingerprint, facial recognition, iris)

•

Unguided AI due to a lack of governance
(limited to ethical principles, system access)

2

•

Cybercrime through
deepfakes, face swapping,
synthetic identities, spam
and robocalls

•

Lack of AI control causes legal risks, e.g.,
organizational fault and trust damages
Lack of AI facilitates scams like fake
identities and abusive prompt injections

3
4

•

Future EUDI Wallet scales weak
biometrics, thus risks “super-hack”

•

Some 60% of all cyber-threats are
identity based, doubling in 2024

Two
challenges
encourage each
other

Ø How to even better secure identities
and distinguish between humans and AI?

•
•

Up to 50 % of all work can be run by
agents and humanoid robots, but “humanin-the-loop” limits potential of AI

Ø How to practically govern digital agents or
humanoid robots so that they do what they
are supposed to do?

* Equals 1013 (coming from USD 3 trillion in 2015)
Source: Statista Market Insights as of 2023 (figure refers to 2025)
- All rights reserved -

3

CGTcoin
The underlying technology trend: The more AI is

autonomous the more Gimel is becoming relevant

1

AI

2025

2028

2

Reactive (RAG, chatbots)

>80%

<25%

Autonomous (agents, robots)

<20%

>75%

3
4

Rise of autonomous AI drives gradual replacement of purely reactive AI applications by
more intelligent, proactive systems like digital agents and humanoid robots.
Particularly in Marketing, Customer Service and Process Automation (not limited to it),
increasing demand for secure agent authorization and ID verification for authorizers.

Source: Estimates based on Deloitte Research Data, McKinsey Technology Trends Outlook, Gartner Hype Cycle for AI amongst others
- All rights reserved -

4

The solution: Gimel provides the open-source platform
for AI control and secure IDs

Gimel Technologies provides the premier operating
system for secure
identities
agent
Gimel
platform (alongand
user life
cycle) control
Gimel ID - Secure identity

G-Agent - Agent control

Are you really a human?

Is the agent really authorized?

Unique selling proposition (USP):
•A Next-level secure biometrics:
Based on DNA data, while
maintaining data privacy patent pending*.

1
2
3

Verify
human
identity

Globally
prove
personhood

A

Authenticate
as human

Authorize
agent / robot

B

•B Global Identity: First and only
AI-based identity proof
(Gimel ID) and continuous risk
monitoring (DefconG) among
up to 8 billion individuals –
patent pending*.

Legitimate as
agent / robot

C

4
•

Verify national
identity (e.g. eID)
as identity
service
provider

•

Enrol DNA based
identity

•

Host identity
within wallet

•

Verify identity
of authorizer

•

Test global
collisions

•

Share with
relying parties

•

Delegate power
of attorney

Run DefconG

•

Leverage
trust services

•

•

- All rights reserved -

Register AI agent
via token

•

Prove agent`s
power of attorney
towards third parties

•

Delegate to /
authorize further
team agents / robots

The Gimel open-source platform integrates into one system for ID-verification, proof of
personhood and authentication of humans as well as authorization and legitimation of AI

* Based on global PCT application at European Patent Office (EPO)
- All rights reserved -

4

• Agent Control: First and only
C
practical AI governance solution
(G-Agent), based on opensource protocol (GAuth) patent pending*.
Ø Cyber-security beyond highassurance (i.e.,
5
beyond LoA4)

CGTcoin
How it works in practice: We operate the Gimel

1
2

Gimel ID

Gimel data flow

Verify

Prove

3
Authenticate

4
G-Agent

Authorize

Legitimate

CA: Certificate Authority
NFC: Near Field Communication
- All rights reserved -

DNA test
(Sequencing Device)
Hashing
(Gimel Device)

QR code
(agents)

Rapid test
(Gimel Dash)

Collision test
(Gimel Cloud)

Gimel Wallet
(incl. DefconG)
Token
(web3)

•

Certificate
(Gimel CA)

ID card
(NFC)

NFT
(web3)

Smart contract
(web3)
NFC
(robots)

Gimel architecture layer (illustrative, AI-enabled)
: Lab system
: Cloud
: Public Key Infrastructure
: Web3 / Blockchain

•

•
•

•

•

Users perform a DNA test,
while no DNA data leave the lab
Quick check-ups of others`
identities based on rapid DNA
test with Gimel Dash (within
some 90 minutes)
DefconG: Users track relevant
threats to identity, globally
If threatened, users revoke their
Gimel ID and renew it

User experience

infrastructure to create a new user experience

Clearly identified users
authorize agents and robots
via the G-Agent, leveraging
GAuth
Third-party users double-check
on agents` legitimation
6

CGTcoin
Use cases: Gimel facilitates various
beneficial use cases for Siemens
•

Potential
Automating
processes

1

•

Delegating tasks

2

•

Improving
scalability

•

Procurement (e.g. demand planer, strategic sourcing,
supplier management)

3

•

Securing quality

•

4

•

Leveraging data
insights

•

Customer service (e.g. customer service
representative)
See deep dive
Marketing (e.g. amplification of market presence and
customer engagement)

•

Re-structuring
towards a slim
organization

•

…

Source: Gimel
- All rights reserved -

•
•

Autonomous AI`s use cases
Supply Chain (e.g. inventory, 3PL relationships,
logistics coordinator)
Operations (e.g. performance, maintenance, quality,
workforce manager) – see https://gimelid.com/
g-agent-in-manufacturing/

•
•

HR (e.g. talent management, coaching, recruiting)
IT (e.g. helpdesk, first and second level support)

•

Management (e.g. executive PA, corporate
communication, investor relations)
…

•

7

CGTcoin
GAuth and OAuth: EntraID and Gimel
complement each other
EntraID vs. Gimel

1

EntraID

Gimel

Specification

OAuth

GAuth

Subject

IT

Agentic AI /
humanoid robots

Object

Access to
resource server

Authorizing
transactions/
decisions/
actions

Rights

Access rights

Power of attorney
(PoA)

Token

Access Token

PoA / Extended
Token

•
•
•

Server

OAuth server

GAuth server

Lead time

60 min.*

Some 60 sec.

•

2
3
4

PoA Definition for AI
Parties
•
•
•

Principal
(organization)
Representative/
authorizer
Authorized AI

•
•
•
•

Type and Scope of
Authorization
Type of authorization
Applicable sectors
Applicable regions
Transactions/
decisions/actions

Requirements
Validity period
• Special conditions
Formal requirements
• Rules for death/inc.
Limits of powers
• Security/compl.
(incl. access rights)
• Jurisdiction & law
Spec. rights/obligations
• Conflict resolution

* Gimel ID requiring accelerated PKI, connected with DefconG
Note: For GAuth, Legal Terms of Gimel Foundation apply, Exclusions are subject to separate licensing by Gimel Technologies
Source: For GAuth, Gimel Foundation, GiFo-RFC0111 and GiFo-RFC0115 (excerpt), For OAuth, IETF, RFC6749
- All rights reserved -

8

CGTcoin
Enforcement: Gimel enforces PoA
compliance through three layers

1
2

Rule based enforcement
Requests being a PoA-subset
(deterministic, signature-like)

3

AI-based enforcement*
Requests in line with trained
compliance (anomaly-focused,
covering implied authority, etc.)

Disclosure
Requests being In line with
relying party`s expectations
(mitigating residual risk)

4
Technical enforcement by Gimel
(controls and mechanisms embedded in the Gimel architecture)

PoA: Power of Attorney
* G-Agent
Note: In line with NIST SP 800-162 and NIST SP 800-207
- All rights reserved -

Legal enforcement by Gimel
(Gimel empowering third party to verify PoA)
9

CGTcoin
AIR: Other agent IDs don`t meet key
requirements like SSI and Agency
AIR requirements (selection)

Gimel

A2A

CSA

IETF

OIDF

Decentral ID / no ANS

✅

O

❌

O

O

1

Distinguish ID from PoA

✅

❌

❌

O

✅

Comprehensive PoA def.

✅

❌

❌

❌

❌

2

No self-authorization

✅

❌

❌

O

O

Disclosure to relying party

✅

❌

❌

❌

❌

PoA as business ownership

✅

❌

❌

❌

❌

Ultimate accountability*

✅

❌

❌

O

O

Least privilege considered

✅

O

O

❌

✅

Solution for agent teams**

✅

O

O

O

O

E2E spec. within protocol

✅

❌

O

❌

❌

Leverage proven safeguards

✅

O

O

✅

✅

3
4

Sel
ect
ion

Evaluation of new whitepapers is ongoing - overall trend: current approaches are not
comprehensively meeting needs for AI identification and authorization
Legend: X: Requirement not met, O: Requirement partly met; V: Requirement met; * Incl. Owner`s Authorizer; ** I.e., lead agent delegates to team agents
AIR: Agent Identity Registration; ID: Identity; Agency: Law of Agency; ANS: Agent Naming Service; PoA: Power of Attorney; OIDF: Open ID Foundation, CSA: Cloud Security Alliance
Source: Internal review of third-party whitepapers versus Gimel protocols (work in progress); A2A: Agent-to-Agent/Google; CSA: Agentic AI IAM; IETF: Wahl, SCIM; OIDF: Identity Management for Agentic AI
10
- All rights reserved -

Biometrics: As eIDAS 2.0 requests the
use of eID, Gimel improves security
1

Security
(ease of
hack)

Hard

DNA
(Gimel ID)

ü High stability over
time
ü Highly regulated
infrastructure

2
3

eID

4

Fingerprint
(e.g., EU passport)

Easy

ü Global PoP can be
reproduced

Iris
(e.g., World ID)

ü Data remain
private, i.e. no
DNA data leaving
the lab

Facial
(e.g., ID.me)

1 false in
400

: Non-inclusive
: Inclusive
PoP: Proof of Personhood
Note: DNA test at 13 Short-Tandem-Repeats
- All rights reserved -

1 false in
1,000

1 false in
40 trillion

1 false in
>100 trillion
Accuracy
(false positive)

11

CGTcoin
Siemens Agent Factory: Gimel enables
practical AI control and secure IDs
GAuth steps

Subject

2
3
4

GAuth authorization protocol

1

I - VIII

i

•
•

EntraID for authorizer*
Agent Identity
Registration (AIR) for
agents

•

Corporate doctor`s office
for sample collection

Subscription
of Parties

Authorization
a - h Request
Handling

Gimel services for …

Siemens
internally

•

GAuth Authorization
Server

Compliance
Tracking & QA

* Based on accelerated PKI, connected with Gimel ID / DefconG; ** DevOps, FinOps, InfOps
CA: Certificate Authority; QA: Quality Assurance
Note: Gimel ID, GAuth+ and G-Agent are subject to separate licensing by Gimel Technologies
- All rights reserved -

Orchestration

SecOps

Dr
dis aft for
cus
sio
n

OtherOps**

•
•
•
•
•

Gimel ID for authorizer of AI
Produced in the Gimel lab (as CA)
DefconG (global risk monitoring)
Hosted in Gimel Cloud
Gimel Wallet for convenience

•
•
•

Facilitation of GAuth through G-Agent
Safeguarding via GAuth+ / web3
Orchestration of AI by G-Agent

•
•

Compliance Tracking via G-Agent (SecOps)
QA of AI`s output via G-Agent (InfOps)

12

CGTcoin
Safeguards: Gimel leverages on proven
security practices where applicable
Gimel safeguards (from beta-testing to full-leverage - excerpt)

1
2
3
4

OAuth 2.0 & OIDC
• Scopes that limit data access
• ID tokens to prove identity
without sharing password
• Token binding to prevent
attackers from stealing login
• Proof key for code exchange
(PKCE) for safer mobile log-on
• Refreshed token as temporary
keys with backup-renewal
• …

OAuth 2.1
• Mandatory PKCE (no implicit
grant type, stop of resource
owner password grants, etc.)
• Exact matching for redirect
URIs
• Secured use of refresh tokens,
tied to specific device or user
• Standardized error handling,
i.e., specific feedback for
developers
• …

OID4VC
(and ETSI, e.g. TS119475)
• Digital signatures to ensure
creds. not tampered with
• Encryption to protect
information within credential
• Selective disclosure, sharing
selected attributes
• Zero knowledge proof for
verifiable credentials
• Non-linkability to protect
user
• …

Other safeguards (e.g., attached to Lab, PKI, Web3, etc.)
Basis: Gimel Foundation, GiFo-RFC0111 and GiFo-RFC0115, including IETF and OIDC references therein
- All rights reserved -

13

CGTcoin
Benefits: Gimel is leapfrogging towards
a new standard in AI governance

1
2
3
4

+
+
+
+

Trustworthy - Performing independent AI governance and embracing tech-sovereignty, thus preventing AI
collapses and conflicts of interest.

Open-source - Enabling transparency and reliability to third-parties through our unique authorization
protocol GAuth (Gimel Foundation, patent pending), thus building trust.

Globally scalable - Facilitating AI control beyond Human-in-the-Loop and at scale (patent pending), thus
leveraging the full potential of AI to act autonomously.

Traceable - Tracing quality, compliance and ID among eight billion humans on earth (patent pending), thus
measuring what you get and leapfrogging towards a new de-facto standard in cyber-security (LoA5).

- All rights reserved -

14

CGTcoin
Example Marketing: Autonomous agents require
comprehensive governance by Gimel
Reactive Marketing Agent
•
1
2

•

3
4
•

Autonomous Marketing Agent

Case: Deploying a Retrieval-Augmented Generation
(RAG) system to supercharge Siemens` market
research.
How it works: RAG system is fed with prompts like
“What are the latest trends in smart manufacturing in
Southeast Asia?” or “What regulatory changes are
anticipated for industrial AI in the European Union?”
The RAG scans internal Siemens reports, public
publications, patents, news feeds, and even social
media discussions among engineers and thought
leaders. It then synthesizes the most relevant insights.

•

Value: Siemens can react swiftly to emerging market
demands, competitor movements, and regulatory
changes. The marketing and product strategy teams
receive automated, evidence-based
recommendations for campaign focus, etc.

•

- All rights reserved -

•

Case: Leveraging an AI-powered digital marketing
agent to amplify presence and engagement across
multiple channels.
How it works: The digital agent analyses trending
topics, audience sentiment, and influencer networks
within the engineering and technology communities. It
autonomously crafts content and interacts in real
time: answering queries from potential clients,
responding to thought leaders, and curating usergenerated content, all while adhering to corporate
messaging guidelines.
Value: This approach ensures that Siemens maintains
a dynamic, responsive, and highly visible brand
presence. The agent not only increases reach and
engagement but also surfaces actionable insights,
adapts messaging proactively and manages budgets.

Example
vendors:

15

CGTcoin
PoA-map: The delegated powers for the

marketing agent are being structured by PoA-maps
ap
-m le
A
Po mp
exa

1
2

Determining PoA credential:
Principal
Parties

Representative
Auth. Client

Price

Sole / joint

PoA
Type & Scope

3
4

Product

Requirement

Restrictions
Delegation

Place

Signature
Sectors
Regions
Actions
Validity
Formalities

Promotion

…

…

PoA-map level 1 – 3:
Generic PoA structure
(abstract RFC0115 attributes)

…
Budget

Features

Social Media

Target group

Quality level

Television

Features

Branding

Print

USP

List price

Outdoor Adv.

Narratives

Discount

Search Engine

Individualize

Credits

Email

Call for Action

Channel
Logistics
Media
...
…
Opex

Influencer
Content M.
Event M.
Mobile M.
Podcasting
Ext. spend

KPIs
Frequency
…
Base
Tools
Performance

…

…

…

PoA-map level 4 – N:
Use-case specific PoA attributes
(example Marketing)

PoA: Power of Attorney; ABPC: Attribute Based Power-of-Attorney Control; RBPC: Role Based Power-of-Attorney Control
- All rights reserved -

•

PoA-map structures PoA
credential via defined attributes
and their values on various
levels

•

PoA credential and PoA-map to
be jointly customized towards
corporate specific policies
and use cases (e.g., Marketing,
Sales, Supply Chain)

•

Deployment of PoA-map
facilitates agent governance at
scale in line with ABPC and
RBPC principles

16

CGTcoin
Collaboration: Overlapping maps of requesting

and relying agent determine space for collaboration
ple
m
a
Ex erlap
ov

Subscription:
•

1
2

•

3
•
4

PoA-map of requesting agent, e.g.,
marketing agent of Siemens
(within GAuth called “Client”)
: Overlap, i.e., collaboration space between agents
PoA: Power of Attorney
- All rights reserved -

PoA-map of relying agent, e.g.,
marketing agent of service provider
(within GAuth called “Resource Server”)

Next to verification of agent
identities (and the ID of its
authorizers), determining each
agent`s PoA
Example: Both agents are
authorized for social media
marketing amongst others
Example code (overlap):
# Powers of Attorney for Agent A
party_a = {
"action": {"post", "edit"},
"platform": {"linkedin", "twitter"},
"region": {"EU"},
"budget_limit": 8000
}
# Powers of Attorney for Agent B
party_b = {
"action": {"edit", "delete"},
"platform": {"linkedin", "facebook"},
"region”å: {"EU", "US"},
"budget_limit": 12000
}
# Find overlap (intersection) on set-type attributes
overlap = {
"action": party_a["action"] & party_b["action"],
"platform": party_a["platform"] & party_b["platform"],
"region": party_a["region"] & party_b["region"],
"budget_limit": min(party_a["budget_limit"], party_b["budget_limit"])
}
print("Collaborative overlap:")
for key, value in overlap.items():
print(f"{key}: {value}")
# Output:
# action: {'edit'}
# platform: {'linkedin'}
# region: {'EU'}
# budget_limit: 8000

17

CGTcoin
Request: The requests of the marketing agent

must be compliant with its general PoA credential
sed t
a
b
en
leRu rcem
o
enf

1
2

Rule-based enforcement:
✅
✅
✅
✅
✅

3
✅

4

Marketing PoA credential
(representing general powers of Client /
Marketing Agent at subscription)

Single request for targeting
(example marketing request in
compliance with general powers)

✅ : Matching, i.e., request being subset of PoA (in other words, each attribute in the transaction must exist in the PoA credential,

and its requested value must be allowed by the credential).
PoA: Power of Attorney
- All rights reserved -

•

For single request, relevant
attributes and their values are
being identified and matched

•

Example: Decision for specific
targeting on social media in line
with policy and budget

•

Example code (match):
# General power of attorney (credential) attributes
credential = {
"action": {"post", "edit", "delete"},
"platform": {"linkedin", "twitter"},
"budget_limit": 10000,
"region": {"EU", "US"}
}
# Incoming request attributes
request = {
"action": "post",
"platform": "linkedin",
"budget": 5000,
"region": "EU"
}
def is_request_authorized(credential, request):
# Check actions
if request["action"] not in credential["action"]:
return False
# Check platforms
if request["platform"] not in credential["platform"]:
return False
# Check budget
if request["budget"] > credential["budget_limit"]:
return False
# Check region
if request["region"] not in credential["region"]:
return False
return True
print(is_request_authorized(credential, request)) # Output: True

18

CGTcoin
Tracking: G-Agent can consider best practices

of Siemens as well as global PoA court-ruling data
sed nt
a
b
AI- ceme
or
enf

1
2
3

AI-based enforcement:
•

For tracking agent`s single
request and behaviour

G-Agent

•

Compliance with PoA
and implied authority

•

Example: Approval for implied
authority
Example prompts (G-Agent):
#Role: Act as a supervisor focused on governing the marketing agent`s
requests and behaviour (transactions, actions, decisions), i.e.,
securing its compliance with the delegated PoA as well as any
implied authority associated

✅

4

Marketing PoA
credential

#Task:
Begin with a concise checklist you will follow up to fulfil this
role, focusing on comprehensiveness rather than details.
Identify and track list of explicitly delegated authorities as
per PoA
Identify and track list of non-explicit, implied authorities
associated with PoA
Ensure each list item offers sufficient evidence for being
compliant
Exclude potentially false positives (accepted but potentially
wrong) as well as obvious false negatives (rejected but
obviously wrong)

Single request for media
advertising

✅ : Compliance confirmed (example, )audit trail documented, opportunities to finetune marketing agent identified.

PoA: Power of Attorney
- All rights reserved -

#Restrictions:
Double check restrictions of PoA like budget constraints
considering so far expenditures
Consider any other requirements (formalities, limits of PoA,
etc.)
#Reasoning
Vet all requests and behaviour to facilitate it is feasible,
professional and in line with best marketing practices of
Siemens
Review all requests and behaviours to ensure that implied
authority complies with relevant court rulings worldwide
#Output
Return the results as properly formatted table, incl. identified
needs to finetune marketing agent as well as “serve the request”
or “do not serve the request” statement

19

CGTcoin
Anomalies: G-Agent enables proper governance

like to conflicts of interest and implied authority
ple s
m
Exa malie
ano

Conflict of interest

Implied authority

1

•

Marketing agent is pre-prompted by agent provider,
which serves competing companies

•

Run contests or giveaways on social media, even if
those specific tactics aren`t outlined, formally

2

•

Marketing agent recommends media outlet, which is
having a financial stake by agent provider

•

Use client`s logos and branding material to run the
client`s social media ads, although explicitly agreed

3

•

•

4

•

Marketing agent prioritizes ist own services over
what is best for the client
Marketing agent (or agent provider, resp.) has a
relationship with a vendor and pushes their products
even if they aren`t the ideal solution
…

Respond to customer inquiries or comments, when
running client`s social media account
Follow relevant accounts and engage with
influencers for growing the client`s followers on
social media
...

•

Ensure agent`s focus is on client success,
not their own interest
- All rights reserved -

•

•

Consider what is reasonably necessary
to achieve the goals
20

CGTcoin
Gimel ID: Verifying the identity of the marketing-

New
globalowner
identity and
network forminimal-invasively
fostering health equity
agent`s
canfinance
be performed
eID
(classic biometrics)
1
2

Gimel ID
(DNA
based)

3
4

Step I – “Must be done anyway”
• Verification of the ID via the digital ID card (eID)
• Sufficient to be EUDI-Wallet compliant
• A ‘must’ for all employees and customers associated with critical infrastructure
Step II – “Secure verification of the agent`s owner”
• ‘Upgrade’ of identity via genetic fingerprint
• Provides premium security beyond high assurance
• Particularly recommended for a security-relevant teams and for customers who
want ‘more security’ (but not limited to it)
ü Stepwise increase security
ü Migrate in a systematic way to a future-proof solution
ü Prepare for improved security in case of future cyber-threats

Example
migration path
at Siemens:
eID: National ID
- All rights reserved -

For now, no plans to introduce Gimel ID across the board at Siemens

21

CGTcoin
Proof of Concept: Setting up five workstream

with a joint team and hitting the ground running
GAuth

G-Agent

GAuth+/Web3

•
•

N.N. (S)
Götz W. (G)

•
•

Christoph P. (G)
N.N. (S)

•

Scope

•
•

GAuth. Server
AIR

•
•

GAuth control
QA of AI Ops

•
•

Sub

•

PKI / Ejbca &
Nexus

•

•
LLM / Mixtral &
Roberta, IT provider

License

•

Open-source*

Team

•
•

Christian F. ,
Kas d.S., Pezh B. (G) •
N.N. (S)

SSI
PoA

•
•
•

Gimel Lab
Gimel Device
Polymorphisms

•
•
•

Gimel Cloud
DefconG
G-Agent front end

Network / Helium
& Iota

•

Forensics Lab of
UKMS

•

IT provider,
White label

3
4

Dra
G-Walletdisc ft for
uss
• Bjørn B. (G) ion

Eicke S. (G)
N.N. (S)

1
2

Gimel ID

N.N. (S)

Proprietary Gimel agreement

The Gimel project to establish both an effective AI governance as well as secure IDs for Siemens and its Agent
Factory is including five PoC workstreams, the 4S-planning and a subsequent global implementation.
* Conditional OS license subject to overall agreement
G: Gimel; S: Siemens; AIR: Agent Identity Registration; SSI: Self-Sovereign-Identity; PoA: Power of Attorney; UKMS: University Clinics of Münster
- All rights reserved -

22

CGTcoin
Proof of Concept: The draft GAuth engine
allows to jump-start the PoC

1
2

-E
GA xcerp
uth
t
eng ine

•

GAuth implementation has
been drafted in a joint team with
exclusive IP ownership of Gimel

•

While ongoing QA is needed,
implementation is supposed to be
compliant with GiFo-RFC0111
and GiFo-RFC0115

•

Gimel may grant Siemens
conditional license rights to
facilitate a jump-start PoC*

3
4

* In the context of a Gimel-project for Siemens, Gimel may grant Siemens rights of use for an implementation Siemens-internally only based on the Legal Provisions of Gimel Foundation (GiFo-RFC0090, etc.)
together with the maintenance of Gimel`s exclusive ownership and considering Exclusions, which are subject to separate licensing. In this case, Siemens grants Gimel the rights of use for any contributions being
made by Siemens or its employees in terms of exclusive, perpetual, transferable, irrevocable, worldwide rights of use.
- All rights reserved -

23

CGTcoin
Product development: Gimel is continuously
driving its innovative solutions

1

Product
development
pipeline

GAuth

G-Agent

(see open-source
specifications on
next page)

•

•

Agent Training
regarding Law of
Agency claims
and beyond
(FZ Jülich)

•

•

Orchestration of
agent portfolios

•

•

AI marketplace

•

QA for Verticals
(e.g., Shopfloor,
Healthcare)

•

…

…

2
3
4

GAuth+/Web3

•

Expanding
node-network
Crypto
incentives
(“Gimel Coin”)
…

Gimel ID

G-Wallet

•

Open-source lab
procedure

•

DefconG
Training

•

Gimel Kit and
Software

•

QeAA alignment
for B2C

•
•

Lab automation
Rapid analyser

•

…

•

Quantum
resistance

•

…

Gimel`s discussions with potential investors to advance the Gimel solutions and further develop the opensource platform for AI control and secure IDs are work-in-progress. Selected pipeline topics may be useful to
consider within the joint project between Gimel and Siemens, e.g., the AIR specification.
AIR: Agent Identity Registration
- All rights reserved -

24

Contact
Gimel Technologies GmbH

Managing Directors:
Bjørn Baunbæk, Dr. Götz G. Wehberg
Chairman of the Board:
Daniel Hartert
Tel.: +4915112462890
Mail: gwehberg@GimelID.com
Web: https://gimelid.com
Github: https://github.com/Gimel-Foundation

Legal notice: All rights are reserved. The brands and concepts of Gimel Technologies GmbH and its affiliated
companies are protected by copyright. The products, procedures and services of Gimel Technologies GmbH and
its affiliated companies are protected by patent law. Discussion is subject to Trade Secret Law.

