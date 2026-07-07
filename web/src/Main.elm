port module Main exposing (main)

import Browser
import Html exposing (Html, aside, button, code, div, h1, h2, h3, header, input, label, li, main_, option, p, pre, section, select, span, text, textarea, ul)
import Html.Attributes exposing (attribute, checked, class, disabled, placeholder, selected, type_, value)
import Html.Events exposing (on, onCheck, onClick, onInput)
import Json.Decode as Decode
import Json.Encode as Encode
import IncidentDomain exposing (nextLifecycle)


port apiRequest : Encode.Value -> Cmd msg
port apiResponse : (Decode.Value -> msg) -> Sub msg


type alias Model =
    { workspace : String, actor : String, actorDraft : String, incidents : List Incident, channels : List Channel, templates : List Template, recipes : List ContextRecipe, agents : List Agent
    , active : Active, posts : List Post, draft : String, replyTo : Maybe Post, replyToBlockId : String, editing : Maybe Post
    , createKind : String, createTitle : String, createSeverity : String, createScope : String, templateId : String, summaryDraft : String, memberDraft : String, memberRole : String, structuredDraft : String
    , structuredType : String, coordination : Coordination, memberships : List Membership, relationships : List IncidentRelationship, relationTarget : String, relationKind : String, splitTitle : String, collections : List ContextCollection, recipePreview : Maybe ContextCollection, recipeId : String, searchQuery : String, searchResults : List KnowledgeSearchResult, similar : List SimilarIncident, agentId : String, agentTask : String, agentRuns : List AgentRun, aiProposals : List AIProposal, workflowDefinitions : List WorkflowDefinition, workflowId : String, workflowRuns : List WorkflowRun, workflowView : String, investigations : List Investigation, busy : Bool, error : Maybe String
    , adminOpen : Bool, adminAgentName : String, adminAgentPurpose : String, adminAgentModel : String, adminTemplateName : String, adminTemplateScope : String, auditEvents : List AuditEvent
    }

type Active = None | IncidentActive Incident | ChannelActive Channel

type alias Incident =
    { id : String, title : String, ownerId : String, severity : String, lifecycle : String
    , scope : List String, verifiedSummary : String, closureChecklist : List ChecklistItem, aiDetected : Bool, configuration : String
    }

type alias ChecklistItem = { id : String, label : String, completed : Bool }
type alias Channel = { id : String, title : String, description : String }
type alias Template = { id : String, name : String, version : Int }
type alias Post = { id : String, authorId : String, revision : Int, replyToPostId : String, replyToBlockId : String, blocks : List Block, createdAt : String }
type alias Block = { id : String, kind : String, body : String }
type MarkdownChunk = Prose String | CodeFence String String
type alias Response = { tag : String, ok : Bool, status : Int, body : Decode.Value }
type alias Coordination = { facts : List Fact, decisions : List Decision, actions : List Action, polls : List Poll, approvals : List Approval }
type alias Fact = { id : String, statement : String, state : String }
type alias Decision = { id : String, statement : String, status : String }
type alias Action = { id : String, title : String, status : String }
type alias Poll = { id : String, question : String, options : List PollOption }
type alias PollOption = { id : String, label : String }
type alias Approval = { id : String, actionId : String, outcome : String }
type alias Membership = { principalId : String, role : String, status : String }
type alias IncidentRelationship = { id : String, sourceIncidentId : String, targetIncidentId : String, kind : String }
type alias KnowledgeSearchResult = { kind : String, incidentId : String, title : String, excerpt : String }
type alias ContextRecipe = { id : String, name : String, version : Int }
type alias ContextCollection = { id : String, status : String, snapshots : List ContextSnapshot, failures : List RetrievalFailure }
type alias ContextSnapshot = { source : String, query : String, retrievedAt : String }
type alias RetrievalFailure = { source : String, category : String, message : String, requiredHumanAction : String }
type alias SimilarIncident = { incident : Incident, score : Int }
type alias Agent = { id : String, name : String, purpose : String, model : String }
type alias AgentRun = { id : String, status : String, task : String, terminationReason : String }
type alias AIProposal = { id : String, kind : String, content : String, status : String, evidenceBlockIds : List String, redacted : Bool }
type alias WorkflowDefinition = { id : String, name : String, version : Int }
type alias WorkflowRun = { id : String, status : String, definitionVersion : Int, steps : List WorkflowStepState }
type alias WorkflowStepState = { stepId : String, name : String, mode : String, risk : String, status : String, stoppedBy : String }
type alias Investigation = { id : String, status : String, repository : String, readOnly : Bool, branch : String, evidence : List InvestigationEvidence, pullRequest : Maybe PullRequest }
type alias InvestigationEvidence = { kind : String, summary : String, sha256 : String }
type alias PullRequest = { number : Int, url : String }
type alias AuditEvent = { actorId : String, action : String, subjectId : String, occurredAt : String }

type Msg
    = GotApi Decode.Value | SelectIncident Incident | SelectChannel Channel | SetDraft String
    | Publish | SetReply Post String | DerivePost Post | EditPost Post | CancelReply | SetCreateKind String | SetCreateTitle String | SetCreateSeverity String | SetCreateScope String | SetTemplate String | CreateIncident | CreateDetectedIncident | CreateChannel
    | AdvanceLifecycle | CancelFalseAlarm | MigrateConfiguration | SetSummary String | SaveSummary | ToggleChecklist ChecklistItem Bool
    | SetActorDraft String | ApplyActor | SetMember String | SetMemberRole String | AddMember | UpdateMember Membership String Bool | TransferOwnership | SetRelationTarget String | SetRelationKind String | RelateIncident | SetSplitTitle String | SplitIncident | SetStructured String | SetStructuredType String | AddStructured
    | DecideItem Decision String | AdvanceAction Action | VotePoll Poll | RequestApproval Action | RespondApproval Approval String | SetRecipe String | PreviewContext | CollectContext | RefreshContext ContextCollection | SetSearchQuery String | SearchKnowledge | SetAgent String | SetAgentTask String | ActivateAgent | RunAgent | ReviewProposal AIProposal String | SetWorkflow String | StartWorkflow | SetWorkflowView String | WorkflowCommand WorkflowRun String String | StartInvestigation | ExecuteInvestigation Investigation String | PrepareInvestigationPatch Investigation | CreateInvestigationPR Investigation | DestroyInvestigation Investigation | DismissError | Reload
    | OpenAdmin | CloseAdmin | SetAdminAgentName String | SetAdminAgentPurpose String | SetAdminAgentModel String | CreateAdminAgent | SetAdminTemplateName String | SetAdminTemplateScope String | CreateAdminTemplate | LoadAudit


init : () -> ( Model, Cmd Msg )
init _ =
    ( { workspace = "acme", actor = "alice", actorDraft = "alice", incidents = [], channels = [], templates = [], recipes = [], agents = [], active = None, posts = []
      , draft = "", replyTo = Nothing, replyToBlockId = "", editing = Nothing, createKind = "incident", createTitle = "", createSeverity = "unclassified", createScope = "", templateId = "", summaryDraft = ""
      , memberDraft = "", memberRole = "participant", structuredDraft = "", structuredType = "fact", coordination = emptyCoordination, memberships = [], relationships = [], relationTarget = "", relationKind = "related", splitTitle = "", collections = [], recipePreview = Nothing, recipeId = "", searchQuery = "", searchResults = [], similar = [], agentId = "", agentTask = "", agentRuns = [], aiProposals = [], workflowDefinitions = [], workflowId = "", workflowRuns = [], workflowView = "checklist", investigations = [], busy = True, error = Nothing
      , adminOpen = False, adminAgentName = "", adminAgentPurpose = "", adminAgentModel = "codex-session", adminTemplateName = "", adminTemplateScope = "", auditEvents = [] }
    , Cmd.batch [ getAs "incidents" "/api/v1/workspaces/acme/incidents" "alice", get "channels" "/api/v1/workspaces/acme/permanent-channels", get "templates" "/api/v1/workspaces/acme/incident-templates", get "recipes" "/api/v1/workspaces/acme/context-recipes", get "agents" "/api/v1/workspaces/acme/agents", get "workflowDefinitions" "/api/v1/workspaces/acme/workflow-definitions" ]
    )


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        GotApi raw -> handleResponse raw model
        SelectIncident incident -> ( { model | active = IncidentActive incident, posts = [], coordination = emptyCoordination, memberships = [], relationships = [], relationTarget = "", splitTitle = "", collections = [], recipePreview = Nothing, similar = [], agentRuns = [], aiProposals = [], workflowRuns = [], investigations = [], summaryDraft = incident.verifiedSummary, error = Nothing }, Cmd.batch [ getAs "posts" (incidentPath model incident.id ++ "/posts") model.actor, getAs "memberships" (incidentPath model incident.id ++ "/members") model.actor, getAs "relationships" (incidentPath model incident.id ++ "/relationships") model.actor, getAs "coordination" (incidentPath model incident.id ++ "/coordination") model.actor, getAs "collections" (incidentPath model incident.id ++ "/context-collections") model.actor, getAs "similar" (incidentPath model incident.id ++ "/similar") model.actor, getAs "agentRuns" (incidentPath model incident.id ++ "/agent-runs") model.actor, getAs "aiProposals" (incidentPath model incident.id ++ "/ai-proposals") model.actor, getAs "workflowRuns" (incidentPath model incident.id ++ "/workflow-runs") model.actor, getAs "investigations" (incidentPath model incident.id ++ "/investigations") model.actor ] )
        SelectChannel channel -> ( { model | active = ChannelActive channel, posts = [], error = Nothing }, get "posts" (channelPath model channel.id ++ "/posts") )
        SetDraft v -> ( { model | draft = v }, Cmd.none )
        SetReply post blockId -> ( { model | replyTo = Just post, replyToBlockId = blockId, editing = Nothing }, Cmd.none )
        EditPost post ->
            case post.blocks of
                first :: _ -> ( { model | editing = Just post, replyTo = Nothing, replyToBlockId = "", draft = first.body }, Cmd.none )
                [] -> ( model, Cmd.none )
        CancelReply -> ( { model | replyTo = Nothing, replyToBlockId = "", editing = Nothing, draft = "" }, Cmd.none )
        SetCreateKind v -> ( { model | createKind = v, createTitle = "" }, Cmd.none )
        SetCreateTitle v -> ( { model | createTitle = v }, Cmd.none )
        SetCreateSeverity v -> ( { model | createSeverity = v }, Cmd.none )
        SetCreateScope v -> ( { model | createScope = v }, Cmd.none )
        SetTemplate v -> ( { model | templateId = v }, Cmd.none )
        SetSummary v -> ( { model | summaryDraft = v }, Cmd.none )
        SetActorDraft v -> ( { model | actorDraft = v }, Cmd.none )
        ApplyActor ->
            let principal = String.trim model.actorDraft in
            if principal == "" || principal == model.actor then
                ( model, Cmd.none )
            else
                ( { model | actor = principal, actorDraft = principal, active = None, incidents = [], posts = [], memberships = [], coordination = emptyCoordination, collections = [], similar = [], agentRuns = [], aiProposals = [], workflowRuns = [], investigations = [], error = Nothing, busy = True }
                , getAs "incidents" (workspacePath model ++ "/incidents") principal
                )
        SetMember v -> ( { model | memberDraft = v }, Cmd.none )
        SetMemberRole v -> ( { model | memberRole = v }, Cmd.none )
        SetRelationTarget v -> ( { model | relationTarget = v }, Cmd.none )
        SetRelationKind v -> ( { model | relationKind = v }, Cmd.none )
        SetSplitTitle v -> ( { model | splitTitle = v }, Cmd.none )
        SetStructured v -> ( { model | structuredDraft = v }, Cmd.none )
        SetStructuredType v -> ( { model | structuredType = v }, Cmd.none )
        SetRecipe v -> ( { model | recipeId = v }, Cmd.none )
        SetSearchQuery v -> ( { model | searchQuery = v }, Cmd.none )
        SetAgent v -> ( { model | agentId = v }, Cmd.none )
        SetAgentTask v -> ( { model | agentTask = v }, Cmd.none )
        OpenAdmin -> ( { model | adminOpen = True, active = None, error = Nothing }, Cmd.batch [ get "agents" (workspacePath model ++ "/agents"), get "templates" (workspacePath model ++ "/incident-templates"), get "audit" (workspacePath model ++ "/audit-export") ] )
        CloseAdmin -> ( { model | adminOpen = False }, Cmd.none )
        SetAdminAgentName v -> ( { model | adminAgentName = v }, Cmd.none )
        SetAdminAgentPurpose v -> ( { model | adminAgentPurpose = v }, Cmd.none )
        SetAdminAgentModel v -> ( { model | adminAgentModel = v }, Cmd.none )
        CreateAdminAgent ->
            if String.trim model.adminAgentName == "" || String.trim model.adminAgentPurpose == "" then ( model, Cmd.none )
            else command model "adminAgent" "POST" (workspacePath model ++ "/agents") (Encode.object [ ( "name", Encode.string model.adminAgentName ), ( "purpose", Encode.string model.adminAgentPurpose ), ( "provider", Encode.string "vertex-ai" ), ( "model", Encode.string model.adminAgentModel ), ( "capabilities", Encode.list Encode.string [ "synthesis" ] ) ])
        SetAdminTemplateName v -> ( { model | adminTemplateName = v }, Cmd.none )
        SetAdminTemplateScope v -> ( { model | adminTemplateScope = v }, Cmd.none )
        CreateAdminTemplate ->
            if String.trim model.adminTemplateName == "" then ( model, Cmd.none )
            else command model "adminTemplate" "POST" (workspacePath model ++ "/incident-templates") (Encode.object [ ( "name", Encode.string model.adminTemplateName ), ( "description", Encode.string "Created in workspace administration" ), ( "defaultSeverity", Encode.string "unclassified" ), ( "defaultScope", Encode.list Encode.string (splitScope model.adminTemplateScope) ), ( "closureChecklist", Encode.list identity [ Encode.object [ ( "id", Encode.string "summary" ), ( "label", Encode.string "Verified summary completed" ), ( "completed", Encode.bool False ) ] ] ) ])
        LoadAudit -> ( { model | busy = True }, get "audit" (workspacePath model ++ "/audit-export") )
        ActivateAgent -> incidentCommand model "POST" "/agent-activations" (Encode.object [ ( "agentId", Encode.string model.agentId ) ])
        RunAgent -> incidentCommand model "POST" "/agent-runs" (Encode.object [ ( "agentId", Encode.string model.agentId ), ( "task", Encode.string model.agentTask ), ( "classification", Encode.string "internal" ), ( "evidenceBlockIds", Encode.list Encode.string (List.concatMap (\post -> List.map .id post.blocks) model.posts) ), ( "requiredCapabilities", Encode.list Encode.string [] ) ])
        ReviewProposal proposal status -> incidentCommand model "PATCH" ("/ai-proposals/" ++ proposal.id) (Encode.object [ ( "status", Encode.string status ) ])
        SetWorkflow v -> ( { model | workflowId = v }, Cmd.none )
        SetWorkflowView v -> ( { model | workflowView = v }, Cmd.none )
        StartWorkflow -> incidentCommand model "POST" "/workflow-runs" (Encode.object [ ( "definitionId", Encode.string model.workflowId ), ( "definitionVersion", Encode.int 0 ), ( "variables", Encode.object [] ) ])
        WorkflowCommand run commandName stepId -> incidentCommand model "POST" ("/workflow-runs/" ++ run.id ++ "/commands") (Encode.object [ ( "command", Encode.string commandName ), ( "stepId", Encode.string stepId ), ( "justification", Encode.string "Operator command from workflow view" ) ])
        StartInvestigation -> incidentCommand model "POST" "/investigations" (Encode.object [ ( "ref", Encode.string "" ), ( "ttlSeconds", Encode.int 3600 ) ])
        ExecuteInvestigation item kind ->
            let
                commandParts =
                    case kind of
                        "diagnostic" -> [ "go", "test", "./..." ]
                        "patch" -> [ "make", "patch" ]
                        "test" -> [ "make", "test" ]
                        _ -> []
                targetUrl = if kind == "browser" then "https://staging.example.com/health" else ""
            in
            incidentCommand model "POST" ("/investigations/" ++ item.id ++ "/executions") (Encode.object [ ( "kind", Encode.string kind ), ( "summary", Encode.string (investigationSummary kind) ), ( "command", Encode.list Encode.string commandParts ), ( "url", Encode.string targetUrl ) ])
        PrepareInvestigationPatch item -> incidentCommand model "POST" ("/investigations/" ++ item.id ++ "/patch") (Encode.object [ ( "branch", Encode.string ("agent/incident-" ++ String.left 8 item.id) ) ])
        CreateInvestigationPR item -> incidentCommand model "POST" ("/investigations/" ++ item.id ++ "/pull-request") (Encode.object [ ( "title", Encode.string "Fix reproduced incident defect" ), ( "body", Encode.string "Verified through the bounded incident investigation workflow." ) ])
        DestroyInvestigation item -> incidentCommand model "POST" ("/investigations/" ++ item.id ++ "/destroy") (Encode.object [])
        CollectContext -> incidentCommand model "POST" "/context-collections" (Encode.object [ ( "recipeId", Encode.string model.recipeId ), ( "labels", Encode.object [] ) ])
        PreviewContext -> command model "contextPreview" "POST" (workspacePath model ++ "/context-recipes/" ++ model.recipeId ++ "/simulate") (Encode.object [ ( "version", Encode.int 0 ), ( "labels", Encode.object [] ) ])
        SearchKnowledge -> command model "knowledgeSearch" "POST" (workspacePath model ++ "/knowledge-search") (Encode.object [ ( "query", Encode.string (String.trim model.searchQuery) ) ])
        RefreshContext collection -> incidentCommand model "POST" ("/context-collections/" ++ collection.id ++ "/refresh") (Encode.object [ ( "labels", Encode.object [] ) ])
        DismissError -> ( { model | error = Nothing }, Cmd.none )
        Reload -> ( { model | busy = True }, Cmd.batch [ getAs "incidents" (workspacePath model ++ "/incidents") model.actor, get "channels" (workspacePath model ++ "/permanent-channels") ] )
        CreateIncident ->
            if String.trim model.createTitle == "" then ( model, Cmd.none )
            else
                let
                    fields = [ ( "title", Encode.string (String.trim model.createTitle) ) ]
                        ++ (if model.templateId == "" then [ ( "severity", Encode.string model.createSeverity ), ( "scope", Encode.list Encode.string (splitScope model.createScope) ) ] else [ ( "templateId", Encode.string model.templateId ) ])
                in command model "mutate" "POST" (workspacePath model ++ "/incidents") (Encode.object fields)
        CreateDetectedIncident ->
            case model.agents of
                detector :: _ ->
                    if String.trim model.createTitle == "" then ( model, Cmd.none )
                    else command model "mutate" "POST" (workspacePath model ++ "/ai-incident-detections") (Encode.object [ ( "detectorId", Encode.string detector.id ), ( "title", Encode.string (String.trim model.createTitle) ), ( "severity", Encode.string model.createSeverity ), ( "trigger", Encode.string "operator-requested detection evaluation" ), ( "rule", Encode.string "compose-demo-detection-policy-v1" ), ( "confidence", Encode.float 0.95 ), ( "confidenceGate", Encode.float 0.8 ), ( "minimumSeverity", Encode.string "SEV-4" ), ( "supportingEvidence", Encode.list Encode.string [ "operator-visible signal: " ++ String.trim model.createTitle ] ) ])
                [] -> ( model, Cmd.none )
        CreateChannel ->
            if String.trim model.createTitle == "" then ( model, Cmd.none )
            else command model "mutate" "POST" (workspacePath model ++ "/permanent-channels") (Encode.object [ ( "title", Encode.string (String.trim model.createTitle) ) ])
        Publish -> publish model
        AdvanceLifecycle ->
            case model.active of
                IncidentActive incident -> command model "incident" "PATCH" (incidentPath model incident.id) (Encode.object [ ( "lifecycle", Encode.string (nextLifecycle incident.lifecycle) ) ])
                _ -> ( model, Cmd.none )
        CancelFalseAlarm ->
            case model.active of
                IncidentActive incident -> command model "incident" "POST" (incidentPath model incident.id ++ "/cancel-false-alarm") (Encode.object [])
                _ -> ( model, Cmd.none )
        MigrateConfiguration ->
            case model.active of
                IncidentActive incident -> command model "incident" "POST" (incidentPath model incident.id ++ "/configuration-migrations") (Encode.object [ ( "templateId", Encode.string model.templateId ), ( "templateVersion", Encode.int 0 ) ])
                _ -> ( model, Cmd.none )
        SaveSummary ->
            case model.active of
                IncidentActive incident -> command model "incident" "PATCH" (incidentPath model incident.id ++ "/resolution") (Encode.object [ ( "verifiedSummary", Encode.string model.summaryDraft ) ])
                _ -> ( model, Cmd.none )
        ToggleChecklist item completed ->
            case model.active of
                IncidentActive incident -> command model "incident" "PATCH" (incidentPath model incident.id ++ "/resolution") (Encode.object [ ( "checklistItemId", Encode.string item.id ), ( "completed", Encode.bool completed ) ])
                _ -> ( model, Cmd.none )
        AddMember ->
            case model.active of
                IncidentActive incident -> command model "mutate" "POST" (incidentPath model incident.id ++ "/members") (Encode.object [ ( "principalId", Encode.string (String.trim model.memberDraft) ), ( "role", Encode.string model.memberRole ) ])
                _ -> ( model, Cmd.none )
        UpdateMember member role revoke ->
            case model.active of
                IncidentActive incident -> command model "mutate" "PATCH" (incidentPath model incident.id ++ "/members/" ++ member.principalId) (Encode.object [ ( "role", Encode.string role ), ( "revoke", Encode.bool revoke ) ])
                _ -> ( model, Cmd.none )
        TransferOwnership ->
            case model.active of
                IncidentActive incident -> command model "incident" "POST" (incidentPath model incident.id ++ "/ownership-transfers") (Encode.object [ ( "newOwnerId", Encode.string (String.trim model.memberDraft) ) ])
                _ -> ( model, Cmd.none )
        RelateIncident -> incidentCommand model "POST" "/relationships" (Encode.object [ ( "targetIncidentId", Encode.string model.relationTarget ), ( "kind", Encode.string model.relationKind ) ])
        SplitIncident -> incidentCommand model "POST" "/split" (Encode.object [ ( "title", Encode.string (String.trim model.splitTitle) ), ( "postIds", Encode.list Encode.string (List.map .id model.posts) ) ])
        DerivePost post ->
            case model.active of
                IncidentActive incident -> command model "mutate" "POST" (incidentPath model incident.id ++ "/posts/" ++ post.id ++ "/derive") (Encode.object [ ( "destinationIncidentId", Encode.string model.relationTarget ) ])
                _ -> ( model, Cmd.none )
        AddStructured -> addStructured model
        DecideItem item status -> incidentCommand model "PATCH" ("/decisions/" ++ item.id) (Encode.object [ ( "status", Encode.string status ) ])
        AdvanceAction item -> incidentCommand model "PATCH" ("/actions/" ++ item.id) (Encode.object [ ( "status", Encode.string (nextActionStatus item.status) ) ])
        VotePoll poll -> case poll.options of
            first :: _ -> incidentCommand model "POST" ("/polls/" ++ poll.id ++ "/votes") (Encode.object [ ( "optionId", Encode.string first.id ) ])
            [] -> ( model, Cmd.none )
        RequestApproval item -> incidentCommand model "POST" "/approvals" (Encode.object [ ( "actionId", Encode.string item.id ), ( "eligibleApproverIds", Encode.list Encode.string [ model.actor ] ), ( "quorum", Encode.int 1 ) ])
        RespondApproval item decision -> incidentCommand model "POST" ("/approvals/" ++ item.id ++ "/responses") (Encode.object [ ( "decision", Encode.string decision ) ])


publish : Model -> ( Model, Cmd Msg )
publish model =
    if String.trim model.draft == "" then ( model, Cmd.none )
    else
        let
            path = case model.active of
                IncidentActive i -> incidentPath model i.id ++ "/posts"
                ChannelActive c -> channelPath model c.id ++ "/posts"
                None -> ""
            replyFields = case model.replyTo of
                Just post ->
                    [ ( "replyToPostId", Encode.string post.id ) ]
                        ++ (if model.replyToBlockId == "" then [] else [ ( "replyToBlockId", Encode.string model.replyToBlockId ) ])
                Nothing -> []
            block =
                Encode.object
                    [ ( "type", Encode.string "markdown" )
                    , ( "schemaVersion", Encode.int 1 )
                    , ( "payload", Encode.object [ ( "text", Encode.string (String.trim model.draft) ) ] )
                    ]

            payload =
                Encode.object (replyFields ++ [ ( "blocks", Encode.list identity [ block ] ) ])

            ( method, target ) =
                case model.editing of
                    Just post -> ( "PUT", path ++ "/" ++ post.id )
                    Nothing -> ( "POST", path )
        in
        if path == "" then ( model, Cmd.none ) else command { model | draft = "", replyTo = Nothing, replyToBlockId = "", editing = Nothing } "post" method target payload


addStructured : Model -> ( Model, Cmd Msg )
addStructured model =
    case model.active of
        IncidentActive incident ->
            let
                base = incidentPath model incident.id
                ( suffix, payload ) =
                    case model.structuredType of
                        "decision" -> ( "/decisions", Encode.object [ ( "statement", Encode.string model.structuredDraft ), ( "rationale", Encode.string "Recorded during manual coordination" ), ( "evidenceBlockIds", Encode.list Encode.string [] ) ] )
                        "action" -> ( "/actions", Encode.object [ ( "title", Encode.string model.structuredDraft ), ( "ownerId", Encode.string model.actor ), ( "kind", Encode.string "manual" ), ( "parameters", Encode.object [] ), ( "verificationCriteria", Encode.string "Owner verifies completion" ) ] )
                        "poll" -> ( "/polls", Encode.object [ ( "question", Encode.string model.structuredDraft ), ( "mode", Encode.string "advisory" ), ( "options", Encode.list Encode.string [ "Yes", "No" ] ), ( "eligibleVoterIds", Encode.list Encode.string [ model.actor ] ), ( "quorum", Encode.int 1 ), ( "allowVoteChanges", Encode.bool True ) ] )
                        _ -> ( "/facts", Encode.object [ ( "statement", Encode.string model.structuredDraft ), ( "evidenceBlockIds", Encode.list Encode.string [] ) ] )
            in
            if String.trim model.structuredDraft == "" then ( model, Cmd.none ) else command { model | structuredDraft = "" } "mutate" "POST" (base ++ suffix) payload
        _ -> ( model, Cmd.none )


handleResponse : Decode.Value -> Model -> ( Model, Cmd Msg )
handleResponse raw model =
    case Decode.decodeValue responseDecoder raw of
        Err err -> ( { model | busy = False, error = Just (Decode.errorToString err) }, Cmd.none )
        Ok response ->
            if not response.ok then ( { model | busy = False, error = Just (errorMessage response) }, Cmd.none )
            else case response.tag of
                "incidents" -> case Decode.decodeValue (Decode.field "items" (Decode.list incidentDecoder)) response.body of
                    Ok items -> ( { model | incidents = items, busy = False }, Cmd.none )
                    Err err -> decodeFailure model err
                "channels" -> case Decode.decodeValue (Decode.field "items" (Decode.list channelDecoder)) response.body of
                    Ok items -> ( { model | channels = items, busy = False }, Cmd.none )
                    Err err -> decodeFailure model err
                "templates" -> case Decode.decodeValue (Decode.field "items" (Decode.list templateDecoder)) response.body of
                    Ok items -> ( { model | templates = items, busy = False }, Cmd.none )
                    Err err -> decodeFailure model err
                "recipes" -> case Decode.decodeValue (Decode.field "items" (Decode.list contextRecipeDecoder)) response.body of
                    Ok items -> ( { model | recipes = items, busy = False }, Cmd.none )
                    Err err -> decodeFailure model err
                "agents" -> case Decode.decodeValue (Decode.field "items" (Decode.list agentDecoder)) response.body of
                    Ok items -> ( { model | agents = items, busy = False }, Cmd.none )
                    Err err -> decodeFailure model err
                "audit" -> case Decode.decodeValue (Decode.field "items" (Decode.list auditEventDecoder)) response.body of
                    Ok items -> ( { model | auditEvents = items, busy = False }, Cmd.none )
                    Err err -> decodeFailure model err
                "adminAgent" -> ( { model | adminAgentName = "", adminAgentPurpose = "", busy = False }, get "agents" (workspacePath model ++ "/agents") )
                "adminTemplate" -> ( { model | adminTemplateName = "", adminTemplateScope = "", busy = False }, get "templates" (workspacePath model ++ "/incident-templates") )
                "agentRuns" -> case Decode.decodeValue (Decode.field "items" (Decode.list agentRunDecoder)) response.body of
                    Ok items -> ( { model | agentRuns = items, busy = False }, Cmd.none )
                    Err err -> decodeFailure model err
                "aiProposals" -> case Decode.decodeValue (Decode.field "items" (Decode.list aiProposalDecoder)) response.body of
                    Ok items -> ( { model | aiProposals = items, busy = False }, Cmd.none )
                    Err err -> decodeFailure model err
                "workflowDefinitions" -> case Decode.decodeValue (Decode.field "items" (Decode.list workflowDefinitionDecoder)) response.body of
                    Ok items -> ( { model | workflowDefinitions = items, busy = False }, Cmd.none )
                    Err err -> decodeFailure model err
                "workflowRuns" -> case Decode.decodeValue (Decode.field "items" (Decode.list workflowRunDecoder)) response.body of
                    Ok items -> ( { model | workflowRuns = items, busy = False }, Cmd.none )
                    Err err -> decodeFailure model err
                "investigations" -> case Decode.decodeValue (Decode.field "items" (Decode.list investigationDecoder)) response.body of
                    Ok items -> ( { model | investigations = items, busy = False }, Cmd.none )
                    Err err -> decodeFailure model err
                "collections" -> case Decode.decodeValue (Decode.field "items" (Decode.list contextCollectionDecoder)) response.body of
                    Ok items -> ( { model | collections = items, busy = False }, Cmd.none )
                    Err err -> decodeFailure model err
                "contextPreview" -> case Decode.decodeValue contextCollectionDecoder response.body of
                    Ok item -> ( { model | recipePreview = Just item, busy = False }, Cmd.none )
                    Err err -> decodeFailure model err
                "knowledgeSearch" -> case Decode.decodeValue (Decode.field "items" (Decode.list knowledgeSearchDecoder)) response.body of
                    Ok items -> ( { model | searchResults = items, busy = False }, Cmd.none )
                    Err err -> decodeFailure model err
                "similar" -> case Decode.decodeValue (Decode.field "items" (Decode.list similarIncidentDecoder)) response.body of
                    Ok items -> ( { model | similar = items, busy = False }, Cmd.none )
                    Err err -> decodeFailure model err
                "posts" -> case Decode.decodeValue (Decode.field "items" (Decode.list postDecoder)) response.body of
                    Ok items -> ( { model | posts = items, busy = False }, Cmd.none )
                    Err err -> decodeFailure model err
                "memberships" -> case Decode.decodeValue (Decode.field "items" (Decode.list membershipDecoder)) response.body of
                    Ok items -> ( { model | memberships = items, busy = False }, Cmd.none )
                    Err err -> decodeFailure model err
                "relationships" -> case Decode.decodeValue (Decode.field "items" (Decode.list relationshipDecoder)) response.body of
                    Ok items -> ( { model | relationships = items, busy = False }, Cmd.none )
                    Err err -> decodeFailure model err
                "coordination" -> case Decode.decodeValue coordinationDecoder response.body of
                    Ok items -> ( { model | coordination = items, busy = False }, Cmd.none )
                    Err err -> decodeFailure model err
                "incident" -> case Decode.decodeValue incidentDecoder response.body of
                    Ok incident -> ( replaceIncident incident { model | active = IncidentActive incident, summaryDraft = incident.verifiedSummary, busy = False }, Cmd.none )
                    Err err -> decodeFailure model err
                "post" -> ( { model | busy = False }, refreshPosts model )
                _ -> ( { model | createTitle = "", memberDraft = "", splitTitle = "", busy = False }, Cmd.batch [ getAs "incidents" (workspacePath model ++ "/incidents") model.actor, get "channels" (workspacePath model ++ "/permanent-channels"), refreshPosts model, refreshMemberships model, refreshRelationships model, refreshCoordination model, refreshCollections model, refreshAgents model, refreshWorkflows model, refreshInvestigations model ] )


replaceIncident : Incident -> Model -> Model
replaceIncident incident model = { model | incidents = List.map (\item -> if item.id == incident.id then incident else item) model.incidents }

decodeFailure model err = ( { model | busy = False, error = Just (Decode.errorToString err) }, Cmd.none )
errorMessage response = Decode.decodeValue (Decode.field "message" Decode.string) response.body |> Result.withDefault ("Request failed with status " ++ String.fromInt response.status)


view : Model -> Html Msg
view model =
    div [ class "app-shell" ]
        [ header [ class "top-bar" ] [ h1 [] [ text "Gimme Context" ], span [ class "workspace-name" ] [ text model.workspace ], div [ class "actor" ] [ label [] [ span [] [ text "Development principal" ], input [ value model.actorDraft, onInput SetActorDraft, attribute "aria-label" "Development principal", attribute "title" "Temporary local identity sent as X-Principal-ID; not a user profile field" ] [] ], button [ onClick ApplyActor, disabled (String.trim model.actorDraft == "" || String.trim model.actorDraft == model.actor) ] [ text "Switch" ] ] ]
        , case model.error of
            Just message ->
                div [ class "error-banner", attribute "role" "alert" ] [ text message, button [ onClick DismissError ] [ text "Dismiss" ] ]

            Nothing ->
                text ""
        , div [ class "workspace" ] [ navigation model, main_ [ class "incident" ] [ if model.adminOpen then administration model else content model ] ]
        ]


navigation model =
    aside [ class "channel-nav", attribute "aria-label" "Channels" ]
        [ button [ class ("admin-nav" ++ (if model.adminOpen then " selected-channel" else "")), onClick OpenAdmin ] [ text "Workspace administration" ]
        , p [ class "eyebrow" ] [ text "INCIDENTS" ], ul [] (List.map (incidentNav model) model.incidents)
        , p [ class "eyebrow" ] [ text "PERMANENT" ], ul [] (List.map (channelNav model) model.channels)
        , div [ class "create-switch", attribute "aria-label" "Channel type" ]
            [ button [ class (if model.createKind == "incident" then "selected" else ""), onClick (SetCreateKind "incident"), attribute "aria-pressed" (if model.createKind == "incident" then "true" else "false") ] [ text "Incident" ]
            , button [ class (if model.createKind == "permanent" then "selected" else ""), onClick (SetCreateKind "permanent"), attribute "aria-pressed" (if model.createKind == "permanent" then "true" else "false") ] [ text "Permanent" ]
            ]
        , label [] [ text (if model.createKind == "incident" then "Incident title" else "Permanent channel title"), input [ value model.createTitle, placeholder (if model.createKind == "incident" then "Checkout incident" else "Checkout operations"), onInput SetCreateTitle ] [] ]
        , if model.createKind == "incident" then div [ class "incident-create-fields" ]
            [ label [] [ text "Incident severity", select [ value model.createSeverity, onChange SetCreateSeverity ] (List.map (choice model.createSeverity) [ "unclassified", "SEV-1", "SEV-2", "SEV-3", "SEV-4" ]) ]
            , label [] [ text "Incident scope", input [ value model.createScope, placeholder "checkout, production", onInput SetCreateScope ] [] ]
            , label [] [ text "Incident template", select [ value model.templateId, onChange SetTemplate ] (option [ value "" ] [ text "Workspace defaults" ] :: List.map (templateChoice model.templateId) model.templates) ]
            ] else p [ class "create-help" ] [ text "Long-lived discussion and reusable operational knowledge, without an incident lifecycle." ]
        , button [ class "primary-action create-action", onClick (if model.createKind == "incident" then CreateIncident else CreateChannel), disabled (model.busy || String.trim model.createTitle == "") ] [ text (if model.createKind == "incident" then "Create incident" else "Create permanent channel") ]
        , if model.createKind == "incident" then button [ class "create-action", onClick CreateDetectedIncident, disabled (model.busy || String.trim model.createTitle == "" || List.isEmpty model.agents) ] [ text "Create AI-detected incident" ] else text ""
        ]

administration model =
    div [ class "admin-page" ]
        [ pageHeader "Workspace administration" "Configuration, governance, and audit" (Just (button [ onClick CloseAdmin ] [ text "Back to channels" ]))
        , div [ class "admin-layout" ]
            [ section [ class "admin-intro" ]
                [ h2 [] [ text "acme workspace" ]
                , p [] [ text "Local in-memory workspace · data resets when the API container is replaced." ]
                , div [ class "notice notice-warning" ] [ Html.strong [] [ text "Development identity" ], p [] [ text "Acting as uses an unverified X-Principal-ID header. OIDC, re-authentication, and workspace-level authorization are not enforced in this build." ] ]
                ]
            , adminStatusGrid
            , section [ class "admin-card admin-wide" ]
                [ div [ class "card-heading" ] [ div [] [ h3 [] [ text "Approved agents" ], p [] [ text "Define an agent before an incident owner activates it." ] ], statusPill "Available" "available" ]
                , div [ class "admin-form" ]
                    [ label [] [ text "Name", input [ value model.adminAgentName, onInput SetAdminAgentName, placeholder "Incident synthesizer" ] [] ]
                    , label [] [ text "Purpose", input [ value model.adminAgentPurpose, onInput SetAdminAgentPurpose, placeholder "Summarize visible incident evidence" ] [] ]
                    , label [] [ text "Model / adapter", input [ value model.adminAgentModel, onInput SetAdminAgentModel ] [] ]
                    , button [ class "primary-action", onClick CreateAdminAgent, disabled (model.busy || String.trim model.adminAgentName == "" || String.trim model.adminAgentPurpose == "") ] [ text "Approve agent" ]
                    ]
                , p [ class "field-help" ] [ text "For local evaluation, use codex-session as the model label. Execution remains evidence-scoped and will report a visible failure unless a model gateway is configured." ]
                , div [ class "record-list" ] (List.map (\agent -> div [ class "record-row" ] [ div [] [ Html.strong [] [ text agent.name ], p [] [ text agent.purpose ] ], span [] [ text agent.model ] ]) model.agents)
                ]
            , section [ class "admin-card admin-wide" ]
                [ div [ class "card-heading" ] [ div [] [ h3 [] [ text "Incident templates" ], p [] [ text "New incidents snapshot the selected immutable template version." ] ], statusPill "Available" "available" ]
                , div [ class "admin-form" ]
                    [ label [] [ text "Template name", input [ value model.adminTemplateName, onInput SetAdminTemplateName, placeholder "Production service incident" ] [] ]
                    , label [] [ text "Default scope", input [ value model.adminTemplateScope, onInput SetAdminTemplateScope, placeholder "checkout, production" ] [] ]
                    , button [ class "primary-action", onClick CreateAdminTemplate, disabled (model.busy || String.trim model.adminTemplateName == "") ] [ text "Publish version 1" ]
                    ]
                , div [ class "record-list" ] (List.map (\item -> div [ class "record-row" ] [ Html.strong [] [ text item.name ], span [] [ text ("Version " ++ String.fromInt item.version) ] ]) model.templates)
                ]
            , section [ class "admin-card admin-wide" ]
                [ div [ class "card-heading" ] [ div [] [ h3 [] [ text "Audit history" ], p [] [ text "Workspace-scoped append-only activity from this in-memory session." ] ], button [ onClick LoadAudit, disabled model.busy ] [ text "Refresh events" ] ]
                , if List.isEmpty model.auditEvents then p [ class "empty-copy" ] [ text "No audit events yet. Creating an agent or template will add one." ] else div [ class "audit-table" ] (List.map viewAuditEvent model.auditEvents)
                ]
            ]
        ]

adminStatusGrid =
    section [ class "status-grid", attribute "aria-label" "Administration capability status" ]
        [ statusCard "Workspace boundary" "Region, data policy, and platform limits require durable workspace storage." "Not implemented"
        , statusCard "Members and roles" "Incident membership exists; workspace-wide membership and immediate read revocation do not." "Not implemented"
        , statusCard "Identity and re-auth" "OIDC claim mapping and fresh authentication are not connected." "Development only"
        , statusCard "Integrations" "Prometheus, Loki, Alertmanager, and GitHub contracts exist but credential registration is environment-managed." "Partial"
        , statusCard "Risk and autonomy" "Workflow risk rules are enforced; workspace and channel policy editors are not available." "Partial"
        ]

statusCard title_ description status =
    div [ class "admin-card status-card" ] [ div [ class "card-heading" ] [ h3 [] [ text title_ ], statusPill status "pending" ], p [] [ text description ] ]

statusPill label_ kind = span [ class ("status-pill status-" ++ kind) ] [ text label_ ]

viewAuditEvent event =
    div [ class "audit-row" ] [ span [ class "audit-time" ] [ text event.occurredAt ], Html.strong [] [ text event.action ], span [] [ text event.actorId ], span [] [ text event.subjectId ] ]

incidentNav model incident = li [ class (if activeId model.active == incident.id then "selected-channel" else "") ] [ button [ onClick (SelectIncident incident) ] [ text (incident.severity ++ " " ++ incident.title) ] ]
channelNav model channel = li [ class (if activeId model.active == channel.id then "selected-channel" else "") ] [ button [ onClick (SelectChannel channel) ] [ text channel.title ] ]
activeId active =
    case active of
        IncidentActive i -> i.id
        ChannelActive c -> c.id
        None -> ""


content model =
    case model.active of
        None -> section [ class "empty-state" ] [ h2 [] [ text "Human incident coordination" ], p [] [ text "Create an incident or permanent channel to begin." ] ]
        ChannelActive channel -> div [] [ pageHeader channel.title "Permanent channel" Nothing, div [ class "incident-layout single" ] [ feed model ] ]
        IncidentActive incident -> div [] [ incidentHeader model incident, div [ class "incident-layout" ] [ feed model, incidentState model incident ] ]

pageHeader title_ kicker action = header [ class "incident-header" ] [ div [] [ p [ class "incident-kicker" ] [ text kicker ], h2 [] [ text title_ ] ], Maybe.withDefault (text "") action ]
incidentHeader model incident =
    pageHeader incident.title (incident.severity ++ " · " ++ incident.lifecycle ++ (if incident.aiDetected then " · AI detected" else ""))
        (Just (div [ class "header-actions" ]
            [ if incident.aiDetected && incident.lifecycle /= "cancelled" then button [ class "danger-action", onClick CancelFalseAlarm, disabled model.busy ] [ text "Cancel false alarm" ] else text ""
            , button [ class "primary-action", onClick AdvanceLifecycle, disabled (model.busy || nextLifecycle incident.lifecycle == incident.lifecycle) ] [ text (if incident.lifecycle == "resolved" then "Resolved" else "Advance to " ++ nextLifecycle incident.lifecycle) ]
            ]))


feed model =
    section [ class "feed", attribute "aria-label" "Chronological feed" ]
        (List.map (viewPostThread model) (threadRoots model.posts) ++ [ composer model ])

threadRoots posts =
    List.filter
        (\post -> post.replyToPostId == "" || not (List.any (\candidate -> candidate.id == post.replyToPostId) posts))
        posts

viewPostThread model post =
    let
        replies = List.filter (\candidate -> candidate.replyToPostId == post.id) model.posts
    in
    div [ class "post-thread" ]
        [ viewPost model post
        , if List.isEmpty replies then
            text ""
          else
            div [ class "post-children" ] (List.map (viewPostThread model) replies)
        ]

viewPost model post =
    Html.article [ class (if post.replyToPostId == "" then "post" else "post post-reply") ]
        [ div [ class "post-meta" ] [ span [ class "author" ] [ text post.authorId ], span [] [ text ("revision " ++ String.fromInt post.revision) ] ]
        , replyContext model post
        , div [] (List.map (viewBlock post) post.blocks)
        , div [] [ button [ class "text-action", onClick (SetReply post "") ] [ text "Reply to post" ], if post.authorId == model.actor then button [ class "text-action edit-action", onClick (EditPost post) ] [ text "Edit" ] else text "", repostAction model post ]
        ]
repostAction model post =
    case model.active of
        IncidentActive _ -> button [ class "text-action", onClick (DerivePost post), disabled (model.relationTarget == "") ] [ text "Repost to selected incident" ]
        _ -> text ""
replyContext model post =
    if post.replyToPostId == "" then
        text ""
    else
        case List.filter (\candidate -> candidate.id == post.replyToPostId) model.posts |> List.head of
            Just parent ->
                div [ class "reply-context" ]
                    [ span [] [ text (if post.replyToBlockId == "" then "Reply to " else "Reply to block by ") ]
                    , Html.strong [] [ text parent.authorId ]
                    , span [] [ text (": " ++ postExcerpt parent post.replyToBlockId) ]
                    ]

            Nothing ->
                div [ class "reply-context" ] [ text "Reply to an earlier post" ]

postExcerpt post blockId =
    let
        candidates =
            if blockId == "" then post.blocks else List.filter (\block -> block.id == blockId) post.blocks
    in
    case candidates of
        first :: _ ->
            if String.length first.body > 90 then String.left 90 first.body ++ "…" else first.body

        [] ->
            "Referenced content"

viewBlock _ block =
    div [ class ("block block-" ++ block.kind) ]
        (blockContent block)

blockContent block =
    case block.kind of
        "markdown" ->
            [ div [ class "markdown-body" ] (List.map viewMarkdownChunk (markdownChunks block.body)) ]

        "code" ->
            [ pre [ class "code-block" ] [ code [] [ text block.body ] ] ]

        "log" ->
            [ pre [ class "code-block log-block" ] [ code [] [ text block.body ] ] ]

        _ ->
            [ span [ class "block-kind" ] [ text block.kind ], p [] [ text block.body ] ]

viewMarkdownChunk chunk =
    case chunk of
        Prose body -> p [] [ text body ]
        CodeFence language body -> pre [ class "code-block", attribute "data-language" language ] [ code [] [ text body ] ]

markdownChunks body =
    markdownLines (String.lines body) False "" [] []

markdownLines remaining inCode language buffered chunks =
    case remaining of
        [] ->
            appendMarkdownChunk inCode language buffered chunks |> List.reverse

        line :: rest ->
            if String.startsWith "```" line then
                if inCode then
                    markdownLines rest False "" [] (appendMarkdownChunk True language buffered chunks)
                else
                    markdownLines rest True (String.trim (String.dropLeft 3 line)) [] (appendMarkdownChunk False "" buffered chunks)
            else
                markdownLines rest inCode language (line :: buffered) chunks

appendMarkdownChunk inCode language buffered chunks =
    let body = buffered |> List.reverse |> String.join "\n" in
    if String.trim body == "" then chunks
    else if inCode then CodeFence language body :: chunks
    else Prose body :: chunks

composer model =
    div [ class "composer" ]
        [ case model.editing of
            Just post ->
                div [ class "replying" ] [ text ("Editing revision " ++ String.fromInt post.revision), button [ onClick CancelReply ] [ text "Cancel" ] ]

            Nothing ->
                case model.replyTo of
                    Just post ->
                        div [ class "replying" ] [ text ("Replying to " ++ (if model.replyToBlockId == "" then "post by " else "block by ") ++ post.authorId), button [ onClick CancelReply ] [ text "Cancel" ] ]

                    Nothing ->
                        text ""
        , label [] [ text "Add to the channel" ], textarea [ placeholder "Share an update…", value model.draft, onInput SetDraft ] []
        , div [ class "composer-actions" ] [ span [ class "composer-help" ] [ text "Markdown · wrap code in ``` fences" ], button [ class "primary-action", onClick Publish, disabled (model.busy || String.trim model.draft == "") ] [ text (if model.editing == Nothing then "Post update" else "Save revision") ] ]
        ]

incidentState model incident =
    aside [ class "state-panel" ]
        [ h2 [] [ text "Incident state" ], stateField "Owner" incident.ownerId, stateField "Scope" (String.join ", " incident.scope), stateField "Configuration" incident.configuration
        , select [ value model.templateId, onChange SetTemplate, attribute "aria-label" "Incident configuration template" ] (option [ value "" ] [ text "Select newer template" ] :: List.map (templateChoice model.templateId) model.templates)
        , button [ onClick MigrateConfiguration, disabled (model.busy || model.templateId == "") ] [ text "Migrate configuration" ]
        , h2 [] [ text "Verified summary" ], textarea [ value model.summaryDraft, onInput SetSummary, placeholder "Outcome and verified impact" ] [], button [ onClick SaveSummary, disabled model.busy ] [ text "Save summary" ]
        , h2 [] [ text "Closure checklist" ], div [] (List.map (checkItem model) incident.closureChecklist)
        , h2 [] [ text "Participants and ownership" ], input [ value model.memberDraft, onInput SetMember, placeholder "Principal ID", attribute "aria-label" "New participant" ] [], select [ value model.memberRole, onChange SetMemberRole, attribute "aria-label" "Participant role" ] (List.map (choice model.memberRole) [ "participant", "editor", "viewer" ]), button [ onClick AddMember, disabled (model.busy || String.trim model.memberDraft == "") ] [ text "Add participant" ], button [ onClick TransferOwnership, disabled (model.busy || String.trim model.memberDraft == "") ] [ text "Transfer ownership" ]
        , div [ class "member-list" ] (List.map (viewMembership model incident) model.memberships)
        , h2 [] [ text "Related incidents" ]
        , select [ value model.relationTarget, onChange SetRelationTarget, attribute "aria-label" "Related incident" ] (option [ value "" ] [ text "Select visible incident" ] :: List.map (incidentRelationChoice incident) model.incidents)
        , select [ value model.relationKind, onChange SetRelationKind, attribute "aria-label" "Relationship type" ] (List.map (choice model.relationKind) [ "related", "duplicate", "caused-by", "follow-up-to", "recurrence-of" ])
        , button [ onClick RelateIncident, disabled (model.busy || model.relationTarget == "") ] [ text "Link incident" ]
        , div [ class "relationship-list" ] (List.map (viewRelationship model incident) model.relationships)
        , input [ value model.splitTitle, onInput SetSplitTitle, placeholder "New incident title", attribute "aria-label" "Split incident title" ] []
        , button [ onClick SplitIncident, disabled (model.busy || String.trim model.splitTitle == "" || List.isEmpty model.posts) ] [ text "Split feed into incident" ]
        , h2 [] [ text "Structured coordination" ], select [ value model.structuredType, onChange SetStructuredType, attribute "aria-label" "Coordination type" ] (List.map (choice model.structuredType) [ "fact", "decision", "action", "poll" ]), textarea [ value model.structuredDraft, onInput SetStructured, placeholder "Statement, task, or question" ] [], button [ onClick AddStructured, disabled model.busy ] [ text "Add" ]
        , contextPanel model
        , agentPanel model
        , workflowPanel model
        , investigationPanel model
        , coordinationView model
        ]

viewMembership model incident member =
    div [ class "member-row" ]
        [ div [] [ Html.strong [] [ text member.principalId ], span [] [ text (" · " ++ member.role ++ " · " ++ member.status) ] ]
        , if member.status /= "active" || member.principalId == incident.ownerId then
            text ""
          else
            div []
                [ button [ class "compact-action", onClick (UpdateMember member (if member.role == "viewer" then "participant" else "viewer") False), disabled model.busy ] [ text (if member.role == "viewer" then "Make participant" else "Make viewer") ]
                , button [ class "danger-action compact-action", onClick (UpdateMember member member.role True), disabled model.busy ] [ text "Revoke" ]
                ]
        ]

incidentRelationChoice active incident =
    if active.id == incident.id then option [ value incident.id, disabled True ] [ text incident.title ] else option [ value incident.id ] [ text incident.title ]

viewRelationship model active relationship =
    let
        otherId = if relationship.sourceIncidentId == active.id then relationship.targetIncidentId else relationship.sourceIncidentId
        title_ = model.incidents |> List.filter (\item -> item.id == otherId) |> List.head |> Maybe.map .title |> Maybe.withDefault "Visible incident"
    in
    div [ class "relationship-row" ] [ Html.strong [] [ text relationship.kind ], span [] [ text (" · " ++ title_) ] ]

investigationPanel model =
    div [ class "investigation-panel" ]
        [ h2 [] [ text "Sandboxed investigation and GitHub remediation" ]
        , button [ onClick StartInvestigation, disabled model.busy ] [ text "Start disposable investigation" ]
        , p [] [ text "Checkout is read-only until a traceable agent/* patch branch is prepared. Browser evidence is restricted to configured staging origins." ]
        , div [] (List.map viewInvestigation model.investigations)
        ]

viewInvestigation item =
    div [ class ("investigation investigation-" ++ item.status) ]
        [ span [ class "block-kind" ] [ text (item.repository ++ " · " ++ item.status) ]
        , p [] [ text ((if item.readOnly then "Read-only checkout" else "Patch workspace " ++ item.branch) ++ " · " ++ String.fromInt (List.length item.evidence) ++ " reproducible evidence blocks") ]
        , div [] (List.map (\e -> p [] [ text (e.kind ++ ": " ++ e.summary ++ " · sha256:" ++ String.left 12 e.sha256) ]) item.evidence)
        , case item.pullRequest of
            Just pr -> p [] [ text ("GitHub pull request #" ++ String.fromInt pr.number ++ " · " ++ pr.url) ]
            Nothing -> text ""
        , div [ class "investigation-actions" ]
            [ button [ onClick (ExecuteInvestigation item "diagnostic"), disabled (item.status == "destroyed" || hasEvidenceKind "diagnostic" item) ] [ text "Reproduce defect" ]
            , button [ onClick (PrepareInvestigationPatch item), disabled (item.status /= "investigating" || not (hasEvidenceKind "diagnostic" item)) ] [ text "Authorise patch workspace" ]
            , button [ onClick (ExecuteInvestigation item "patch"), disabled (item.status /= "patching" || hasEvidenceKind "patch" item) ] [ text "Apply patch" ]
            , button [ onClick (ExecuteInvestigation item "test"), disabled (item.status /= "patching" || not (hasEvidenceKind "patch" item) || hasEvidenceKind "test" item) ] [ text "Verify patch" ]
            , button [ onClick (ExecuteInvestigation item "browser"), disabled (item.status /= "patching" || not (hasEvidenceKind "test" item) || hasEvidenceKind "browser" item) ] [ text "Capture staging evidence" ]
            , button [ class "primary-action", onClick (CreateInvestigationPR item), disabled (item.status /= "patching" || not (hasEvidenceKind "patch" item) || not (hasEvidenceKind "test" item)) ] [ text "Create traceable pull request" ]
            , button [ class "danger-action", onClick (DestroyInvestigation item), disabled (item.status == "destroyed") ] [ text "Destroy sandbox" ]
            ]
        ]

hasEvidenceKind kind item = List.any (\e -> e.kind == kind) item.evidence

investigationSummary kind =
    case kind of
        "diagnostic" -> "Reproduced the reported defect"
        "patch" -> "Applied the isolated remediation"
        "test" -> "Regression verification passed"
        "browser" -> "Captured allowed staging evidence"
        _ -> "Investigation evidence"

workflowPanel model =
    div [ class "workflow-panel" ]
        [ h2 [] [ text "Workflow and controlled autonomy" ]
        , select [ value model.workflowId, onChange SetWorkflow, attribute "aria-label" "Workflow definition" ] (option [ value "" ] [ text "Select published workflow" ] :: List.map (\d -> option [ value d.id ] [ text (d.name ++ " v" ++ String.fromInt d.version) ]) model.workflowDefinitions)
        , button [ class "primary-action", onClick StartWorkflow, disabled (model.busy || model.workflowId == "") ] [ text "Start workflow" ]
        , select [ value model.workflowView, onChange SetWorkflowView, attribute "aria-label" "Workflow projection" ] (List.map (choice model.workflowView) [ "checklist", "flow" ])
        , div [] (List.map (viewWorkflowRun model.workflowView) model.workflowRuns)
        ]

viewWorkflowRun projection run =
    div [ class ("workflow-run workflow-" ++ projection) ]
        ([ p [] [ text ("Run v" ++ String.fromInt run.definitionVersion ++ " · " ++ run.status) ] ]
            ++ List.map (viewWorkflowStep run) run.steps
            ++ [ div [ class "workflow-actions" ]
                    [ button [ class "secondary-action", onClick (WorkflowCommand run (if run.status == "paused" then "resume" else "pause") ""), disabled (run.status == "completed" || run.status == "cancelled") ] [ text (if run.status == "paused" then "Resume workflow" else "Pause workflow") ]
                    , button [ class "danger-action", onClick (WorkflowCommand run "stop" ""), disabled (run.status == "completed" || run.status == "cancelled") ] [ text "Stop workflow" ]
                    ]
               ]
        )

viewWorkflowStep run step =
    div [ class ("workflow-step step-" ++ step.status) ]
        [ span [ class "block-kind" ] [ text step.name ], span [] [ text (step.mode ++ " · " ++ step.risk ++ " · " ++ step.status) ]
        , if step.status == "pending" then button [ class "primary-action compact-action", onClick (WorkflowCommand run "start-step" step.stepId) ] [ text "Start step" ] else text ""
        , if step.status == "in-progress" || step.status == "countdown" then button [ class "primary-action compact-action", onClick (WorkflowCommand run "complete-step" step.stepId) ] [ text "Complete step" ] else text ""
        , if step.mode == "autonomous" && (step.status == "in-progress" || step.status == "countdown") then button [ onClick (WorkflowCommand run "stop-autonomy" step.stepId) ] [ text "Stop autonomy" ] else text ""
        , if step.status == "stopped" then button [ onClick (WorkflowCommand run "restart-autonomy" step.stepId) ] [ text "Authorised restart" ] else text ""
        , if step.status == "failed" then button [ onClick (WorkflowCommand run "retry-step" step.stepId) ] [ text "Retry" ] else text ""
        ]

agentPanel model =
    div [ class "agent-panel" ]
        [ h2 [] [ text "AI-assisted synthesis" ]
        , select [ value model.agentId, onChange SetAgent, attribute "aria-label" "Incident agent" ] (option [ value "" ] [ text "Select approved agent" ] :: List.map (\a -> option [ value a.id ] [ text (a.name ++ " · " ++ a.model) ]) model.agents)
        , button [ onClick ActivateAgent, disabled (model.busy || model.agentId == "") ] [ text "Activate agent" ]
        , textarea [ value model.agentTask, onInput SetAgentTask, placeholder "Synthesis task using visible feed evidence", attribute "aria-label" "Agent task" ] []
        , button [ onClick RunAgent, disabled (model.busy || model.agentId == "" || String.trim model.agentTask == "" || List.isEmpty model.posts) ] [ text "Run synthesis" ]
        , div [] (List.map (\run -> p [ class ("agent-run run-" ++ run.status) ] [ text (run.task ++ " · " ++ run.status ++ (if run.terminationReason == "" then "" else " · " ++ run.terminationReason)) ]) model.agentRuns)
        , div [] (List.map viewAIProposal model.aiProposals)
        ]

viewAIProposal proposal =
    coordinationItem ("AI " ++ proposal.kind) proposal.content (proposal.status ++ " · " ++ String.fromInt (List.length proposal.evidenceBlockIds) ++ " evidence links" ++ (if proposal.redacted then " · redacted" else "")) (if proposal.status == "proposed" then [ button [ onClick (ReviewProposal proposal "accepted") ] [ text "Accept AI proposal" ], button [ onClick (ReviewProposal proposal "rejected") ] [ text "Reject AI proposal" ] ] else [])

contextPanel model =
    div [ class "context-panel" ]
        [ h2 [] [ text "Operational context" ]
        , select [ value model.recipeId, onChange SetRecipe, attribute "aria-label" "Context recipe" ] (option [ value "" ] [ text "Select recipe" ] :: List.map contextRecipeChoice model.recipes)
        , div [ class "context-actions" ] [ button [ onClick PreviewContext, disabled (model.busy || model.recipeId == "") ] [ text "Preview recipe" ], button [ onClick CollectContext, disabled (model.busy || model.recipeId == "") ] [ text "Collect context" ] ]
        , case model.recipePreview of
            Just preview -> div [ class "context-preview" ] [ Html.strong [] [ text "Preview only — not published" ], p [] [ text (String.fromInt (List.length preview.snapshots) ++ " planned operations") ], div [] (List.map (\snapshot -> p [] [ text (snapshot.source ++ ": " ++ snapshot.query) ]) preview.snapshots) ]
            Nothing -> text ""
        , div [] (List.map viewCollection model.collections)
        , h2 [] [ text "Search platform knowledge" ]
        , input [ value model.searchQuery, onInput SetSearchQuery, placeholder "Search visible incident knowledge", attribute "aria-label" "Knowledge search" ] []
        , button [ onClick SearchKnowledge, disabled (model.busy || String.trim model.searchQuery == "") ] [ text "Search knowledge" ]
        , div [ class "search-results" ] (List.map (\item -> div [ class "search-result" ] [ Html.strong [] [ text (item.kind ++ " · " ++ item.title) ], p [] [ text item.excerpt ] ]) model.searchResults)
        , h2 [] [ text "Similar incidents" ]
        , div [] (List.map (\item -> p [ class "similar-incident" ] [ text (item.incident.severity ++ " " ++ item.incident.title ++ " · score " ++ String.fromInt item.score) ]) model.similar)
        ]

viewCollection item =
    div [ class ("context-collection context-" ++ item.status) ]
        [ span [ class "block-kind" ] [ text ("Collection " ++ item.status) ]
        , p [] [ text (String.fromInt (List.length item.snapshots) ++ " snapshots · " ++ String.fromInt (List.length item.failures) ++ " failures") ]
        , div [] (List.map (\snapshot -> p [] [ text (snapshot.source ++ ": " ++ snapshot.query) ]) item.snapshots)
        , div [] (List.map (\failure -> p [ class "retrieval-failure" ] [ text (failure.source ++ " " ++ failure.category ++ ": " ++ failure.message ++ ". " ++ failure.requiredHumanAction) ]) item.failures)
        , button [ onClick (RefreshContext item) ] [ text "Refresh" ]
        ]

coordinationView model =
    div [ class "coordination-items" ]
        (List.map (\item -> coordinationItem "Fact" item.statement item.state []) model.coordination.facts
            ++ List.map (\item -> coordinationItem "Decision" item.statement item.status (if item.status == "proposed" then [ button [ onClick (DecideItem item "accepted") ] [ text "Accept" ], button [ onClick (DecideItem item "rejected") ] [ text "Reject" ] ] else [])) model.coordination.decisions
            ++ List.map (\item -> coordinationItem "Action" item.title item.status [ button [ onClick (AdvanceAction item), disabled (nextActionStatus item.status == item.status) ] [ text "Advance" ], button [ onClick (RequestApproval item) ] [ text "Request approval" ] ]) model.coordination.actions
            ++ List.map (\item -> coordinationItem "Poll" item.question (String.fromInt (List.length item.options) ++ " options") [ button [ onClick (VotePoll item) ] [ text "Vote first option" ] ]) model.coordination.polls
            ++ List.map (\item -> coordinationItem "Approval" ("Action " ++ item.actionId) item.outcome (if item.outcome == "pending" then [ button [ onClick (RespondApproval item "approve") ] [ text "Approve" ], button [ onClick (RespondApproval item "reject") ] [ text "Reject" ] ] else [])) model.coordination.approvals
        )

coordinationItem kind title_ status actions =
    div [ class "coordination-item" ] [ span [ class "block-kind" ] [ text kind ], p [] [ text title_ ], span [] [ text status ], div [] actions ]
stateField name content_ = div [ class "state-field" ] [ span [] [ text name ], Html.strong [] [ text content_ ] ]
checkItem model item = label [ class "check-item" ] [ input [ type_ "checkbox", checked item.completed, onCheck (ToggleChecklist item), disabled model.busy ] [], text item.label ]
choice current item = option [ value item, selected (current == item) ] [ text item ]
templateChoice current item = option [ value item.id, selected (current == item.id) ] [ text (item.name ++ " v" ++ String.fromInt item.version) ]
contextRecipeChoice item = option [ value item.id ] [ text (item.name ++ " v" ++ String.fromInt item.version) ]
onChange tagger = on "change" (Decode.map tagger (Decode.at [ "target", "value" ] Decode.string))


workspacePath model = "/api/v1/workspaces/" ++ model.workspace
incidentPath model id = workspacePath model ++ "/incidents/" ++ id
channelPath model id = workspacePath model ++ "/permanent-channels/" ++ id

get tag url = apiRequest (Encode.object [ ( "tag", Encode.string tag ), ( "method", Encode.string "GET" ), ( "url", Encode.string url ) ])
getAs tag url actorId = apiRequest (Encode.object [ ( "tag", Encode.string tag ), ( "method", Encode.string "GET" ), ( "url", Encode.string url ), ( "actor", Encode.string actorId ) ])
command model tag method url body = ( { model | busy = True, error = Nothing }, apiRequest (Encode.object [ ( "tag", Encode.string tag ), ( "method", Encode.string method ), ( "url", Encode.string url ), ( "actor", Encode.string model.actor ), ( "body", body ) ]) )
incidentCommand model method suffix body =
    case model.active of
        IncidentActive incident -> command model "mutate" method (incidentPath model incident.id ++ suffix) body
        _ -> ( model, Cmd.none )

nextActionStatus current =
    case current of
        "proposed" -> "ready"
        "ready" -> "in-progress"
        "in-progress" -> "verification"
        "verification" -> "completed"
        _ -> current

emptyCoordination = Coordination [] [] [] [] []
splitScope value_ = value_ |> String.split "," |> List.map String.trim |> List.filter (\item -> item /= "")
refreshPosts model =
    case model.active of
        IncidentActive i -> getAs "posts" (incidentPath model i.id ++ "/posts") model.actor
        ChannelActive c -> get "posts" (channelPath model c.id ++ "/posts")
        None -> Cmd.none
refreshCoordination model =
    case model.active of
        IncidentActive i -> getAs "coordination" (incidentPath model i.id ++ "/coordination") model.actor
        _ -> Cmd.none
refreshMemberships model =
    case model.active of
        IncidentActive i -> getAs "memberships" (incidentPath model i.id ++ "/members") model.actor
        _ -> Cmd.none
refreshRelationships model =
    case model.active of
        IncidentActive i -> getAs "relationships" (incidentPath model i.id ++ "/relationships") model.actor
        _ -> Cmd.none
refreshCollections model =
    case model.active of
        IncidentActive i -> getAs "collections" (incidentPath model i.id ++ "/context-collections") model.actor
        _ -> Cmd.none
refreshAgents model =
    case model.active of
        IncidentActive i -> Cmd.batch [ getAs "agentRuns" (incidentPath model i.id ++ "/agent-runs") model.actor, getAs "aiProposals" (incidentPath model i.id ++ "/ai-proposals") model.actor ]
        _ -> Cmd.none
refreshWorkflows model =
    case model.active of
        IncidentActive i -> getAs "workflowRuns" (incidentPath model i.id ++ "/workflow-runs") model.actor
        _ -> Cmd.none
refreshInvestigations model =
    case model.active of
        IncidentActive i -> getAs "investigations" (incidentPath model i.id ++ "/investigations") model.actor
        _ -> Cmd.none

responseDecoder = Decode.map4 Response (Decode.field "tag" Decode.string) (Decode.field "ok" Decode.bool) (Decode.field "status" Decode.int) (Decode.field "body" Decode.value)
incidentDecoder =
    Decode.map8
        (\id title ownerId severity lifecycle scope verifiedSummary details ->
            { id = id, title = title, ownerId = ownerId, severity = severity, lifecycle = lifecycle, scope = scope, verifiedSummary = verifiedSummary, closureChecklist = details.checklist, aiDetected = details.detected, configuration = details.configuration }
        )
        (Decode.field "id" Decode.string)
        (Decode.field "title" Decode.string)
        (Decode.field "ownerId" Decode.string)
        (Decode.field "severity" Decode.string)
        (Decode.field "lifecycle" Decode.string)
        (Decode.field "scope" (Decode.list Decode.string))
        (Decode.oneOf [ Decode.field "verifiedSummary" Decode.string, Decode.succeed "" ])
        (Decode.map3 (\checklist detected configuration -> { checklist = checklist, detected = detected, configuration = configuration })
            (Decode.field "closureChecklist" (Decode.list checklistDecoder))
            (Decode.oneOf [ Decode.field "detection" Decode.value |> Decode.map (\_ -> True), Decode.succeed False ])
            (Decode.map2 (\snapshot historyCount -> snapshot ++ " · " ++ String.fromInt historyCount ++ " prior snapshots")
                (Decode.oneOf [ Decode.field "templateSnapshot" (Decode.map2 (\name version -> name ++ " v" ++ String.fromInt version) (Decode.field "name" Decode.string) (Decode.field "version" Decode.int)), Decode.succeed "Workspace defaults" ])
                (Decode.oneOf [ Decode.field "configurationHistory" (Decode.list Decode.value) |> Decode.map List.length, Decode.succeed 0 ]))
        )
checklistDecoder = Decode.map3 ChecklistItem (Decode.field "id" Decode.string) (Decode.field "label" Decode.string) (Decode.field "completed" Decode.bool)
channelDecoder = Decode.map3 Channel (Decode.field "id" Decode.string) (Decode.field "title" Decode.string) (Decode.oneOf [ Decode.field "description" Decode.string, Decode.succeed "" ])
templateDecoder = Decode.map3 Template (Decode.field "id" Decode.string) (Decode.field "name" Decode.string) (Decode.field "version" Decode.int)
postDecoder = Decode.map7 Post (Decode.field "id" Decode.string) (Decode.field "authorId" Decode.string) (Decode.field "revision" Decode.int) (Decode.oneOf [ Decode.field "replyToPostId" Decode.string, Decode.succeed "" ]) (Decode.oneOf [ Decode.field "replyToBlockId" Decode.string, Decode.succeed "" ]) (Decode.field "blocks" (Decode.list blockDecoder)) (Decode.field "createdAt" Decode.string)
blockDecoder = Decode.map3 Block (Decode.field "id" Decode.string) (Decode.field "type" Decode.string) (Decode.oneOf [ Decode.at [ "payload", "text" ] Decode.string, Decode.at [ "payload", "label" ] Decode.string, Decode.succeed "Structured block" ])
coordinationDecoder = Decode.map5 Coordination (Decode.field "facts" (Decode.list factDecoder)) (Decode.field "decisions" (Decode.list decisionDecoder)) (Decode.field "actions" (Decode.list actionDecoder)) (Decode.field "polls" (Decode.list pollDecoder)) (Decode.field "approvals" (Decode.list approvalDecoder))
factDecoder = Decode.map3 Fact (Decode.field "id" Decode.string) (Decode.field "statement" Decode.string) (Decode.field "state" Decode.string)
decisionDecoder = Decode.map3 Decision (Decode.field "id" Decode.string) (Decode.field "statement" Decode.string) (Decode.field "status" Decode.string)
actionDecoder = Decode.map3 Action (Decode.field "id" Decode.string) (Decode.field "title" Decode.string) (Decode.field "status" Decode.string)
pollDecoder = Decode.map3 Poll (Decode.field "id" Decode.string) (Decode.field "question" Decode.string) (Decode.field "options" (Decode.list pollOptionDecoder))
pollOptionDecoder = Decode.map2 PollOption (Decode.field "id" Decode.string) (Decode.field "label" Decode.string)
approvalDecoder = Decode.map3 Approval (Decode.field "id" Decode.string) (Decode.field "actionId" Decode.string) (Decode.field "outcome" Decode.string)
membershipDecoder = Decode.map3 Membership (Decode.field "principalId" Decode.string) (Decode.field "role" Decode.string) (Decode.field "status" Decode.string)
relationshipDecoder = Decode.map4 IncidentRelationship (Decode.field "id" Decode.string) (Decode.field "sourceIncidentId" Decode.string) (Decode.field "targetIncidentId" Decode.string) (Decode.field "kind" Decode.string)
knowledgeSearchDecoder = Decode.map4 KnowledgeSearchResult (Decode.field "kind" Decode.string) (Decode.field "incidentId" Decode.string) (Decode.field "title" Decode.string) (Decode.field "excerpt" Decode.string)
contextRecipeDecoder = Decode.map3 ContextRecipe (Decode.field "id" Decode.string) (Decode.field "name" Decode.string) (Decode.field "version" Decode.int)
contextCollectionDecoder = Decode.map4 ContextCollection (Decode.field "id" Decode.string) (Decode.field "status" Decode.string) (Decode.field "snapshots" (Decode.list contextSnapshotDecoder)) (Decode.field "failures" (Decode.list retrievalFailureDecoder))
contextSnapshotDecoder = Decode.map3 ContextSnapshot (Decode.field "source" Decode.string) (Decode.field "query" Decode.string) (Decode.field "retrievedAt" Decode.string)
retrievalFailureDecoder = Decode.map4 RetrievalFailure (Decode.field "source" Decode.string) (Decode.field "category" Decode.string) (Decode.field "message" Decode.string) (Decode.field "requiredHumanAction" Decode.string)
similarIncidentDecoder = Decode.map2 SimilarIncident (Decode.field "incident" incidentDecoder) (Decode.field "score" Decode.int)
agentDecoder = Decode.map4 Agent (Decode.field "id" Decode.string) (Decode.field "name" Decode.string) (Decode.field "purpose" Decode.string) (Decode.field "model" Decode.string)
agentRunDecoder = Decode.map4 AgentRun (Decode.field "id" Decode.string) (Decode.field "status" Decode.string) (Decode.field "task" Decode.string) (Decode.oneOf [ Decode.field "terminationReason" Decode.string, Decode.succeed "" ])
aiProposalDecoder = Decode.map6 AIProposal (Decode.field "id" Decode.string) (Decode.field "kind" Decode.string) (Decode.field "content" Decode.string) (Decode.field "status" Decode.string) (Decode.field "evidenceBlockIds" (Decode.list Decode.string)) (Decode.field "redacted" Decode.bool)
workflowDefinitionDecoder = Decode.map3 WorkflowDefinition (Decode.field "id" Decode.string) (Decode.field "name" Decode.string) (Decode.field "version" Decode.int)
workflowRunDecoder = Decode.map4 WorkflowRun (Decode.field "id" Decode.string) (Decode.field "status" Decode.string) (Decode.field "definitionVersion" Decode.int) (Decode.field "steps" (Decode.list workflowStepStateDecoder))
workflowStepStateDecoder = Decode.map6 WorkflowStepState (Decode.field "stepId" Decode.string) (Decode.field "name" Decode.string) (Decode.field "mode" Decode.string) (Decode.field "risk" Decode.string) (Decode.field "status" Decode.string) (Decode.oneOf [ Decode.field "stoppedBy" Decode.string, Decode.succeed "" ])
investigationDecoder = Decode.map7 Investigation (Decode.field "id" Decode.string) (Decode.field "status" Decode.string) (Decode.field "repository" Decode.string) (Decode.field "readOnly" Decode.bool) (Decode.oneOf [ Decode.field "branch" Decode.string, Decode.succeed "" ]) (Decode.field "evidence" (Decode.list investigationEvidenceDecoder)) (Decode.maybe (Decode.field "pullRequest" pullRequestDecoder))
investigationEvidenceDecoder = Decode.map3 InvestigationEvidence (Decode.field "kind" Decode.string) (Decode.field "summary" Decode.string) (Decode.field "sha256" Decode.string)
pullRequestDecoder = Decode.map2 PullRequest (Decode.field "number" Decode.int) (Decode.field "url" Decode.string)
auditEventDecoder = Decode.map4 AuditEvent (Decode.field "actorId" Decode.string) (Decode.field "action" Decode.string) (Decode.field "targetId" Decode.string) (Decode.field "at" Decode.string)

subscriptions _ = apiResponse GotApi
main = Browser.element { init = init, update = update, subscriptions = subscriptions, view = view }
