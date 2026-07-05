port module Main exposing (main)

import Browser
import Html exposing (Html, aside, button, div, h1, h2, header, input, label, li, main_, option, p, section, select, span, text, textarea, ul)
import Html.Attributes exposing (attribute, checked, class, disabled, placeholder, selected, type_, value)
import Html.Events exposing (on, onCheck, onClick, onInput)
import Json.Decode as Decode
import Json.Encode as Encode
import IncidentDomain exposing (nextLifecycle)


port apiRequest : Encode.Value -> Cmd msg
port apiResponse : (Decode.Value -> msg) -> Sub msg


type alias Model =
    { workspace : String, actor : String, incidents : List Incident, channels : List Channel, templates : List Template, recipes : List ContextRecipe
    , active : Active, posts : List Post, draft : String, blockType : String, replyTo : Maybe Post, editing : Maybe Post
    , createTitle : String, createSeverity : String, createScope : String, templateId : String, summaryDraft : String, memberDraft : String, memberRole : String, structuredDraft : String
    , structuredType : String, coordination : Coordination, collections : List ContextCollection, recipeId : String, similar : List SimilarIncident, busy : Bool, error : Maybe String
    }

type Active = None | IncidentActive Incident | ChannelActive Channel

type alias Incident =
    { id : String, title : String, ownerId : String, severity : String, lifecycle : String
    , scope : List String, verifiedSummary : String, closureChecklist : List ChecklistItem
    }

type alias ChecklistItem = { id : String, label : String, completed : Bool }
type alias Channel = { id : String, title : String, description : String }
type alias Template = { id : String, name : String, version : Int }
type alias Post = { id : String, authorId : String, revision : Int, replyToPostId : String, blocks : List Block, createdAt : String }
type alias Block = { id : String, kind : String, body : String }
type alias Response = { tag : String, ok : Bool, status : Int, body : Decode.Value }
type alias Coordination = { facts : List Fact, decisions : List Decision, actions : List Action, polls : List Poll, approvals : List Approval }
type alias Fact = { id : String, statement : String, state : String }
type alias Decision = { id : String, statement : String, status : String }
type alias Action = { id : String, title : String, status : String }
type alias Poll = { id : String, question : String, options : List PollOption }
type alias PollOption = { id : String, label : String }
type alias Approval = { id : String, actionId : String, outcome : String }
type alias ContextRecipe = { id : String, name : String, version : Int }
type alias ContextCollection = { id : String, status : String, snapshots : List ContextSnapshot, failures : List RetrievalFailure }
type alias ContextSnapshot = { source : String, query : String, retrievedAt : String }
type alias RetrievalFailure = { source : String, category : String, message : String, requiredHumanAction : String }
type alias SimilarIncident = { incident : Incident, score : Int }

type Msg
    = GotApi Decode.Value | SelectIncident Incident | SelectChannel Channel | SetDraft String | SetBlockType String
    | Publish | SetReply Post | EditPost Post | CancelReply | SetCreateTitle String | SetCreateSeverity String | SetCreateScope String | SetTemplate String | CreateIncident | CreateChannel
    | AdvanceLifecycle | SetSummary String | SaveSummary | ToggleChecklist ChecklistItem Bool
    | SetActor String | SetMember String | SetMemberRole String | AddMember | TransferOwnership | SetStructured String | SetStructuredType String | AddStructured
    | DecideItem Decision String | AdvanceAction Action | VotePoll Poll | RequestApproval Action | RespondApproval Approval String | SetRecipe String | CollectContext | RefreshContext ContextCollection | DismissError | Reload


init : () -> ( Model, Cmd Msg )
init _ =
    ( { workspace = "acme", actor = "alice", incidents = [], channels = [], templates = [], recipes = [], active = None, posts = []
      , draft = "", blockType = "markdown", replyTo = Nothing, editing = Nothing, createTitle = "", createSeverity = "unclassified", createScope = "", templateId = "", summaryDraft = ""
      , memberDraft = "", memberRole = "participant", structuredDraft = "", structuredType = "fact", coordination = emptyCoordination, collections = [], recipeId = "", similar = [], busy = True, error = Nothing }
    , Cmd.batch [ get "incidents" "/api/v1/workspaces/acme/incidents", get "channels" "/api/v1/workspaces/acme/permanent-channels", get "templates" "/api/v1/workspaces/acme/incident-templates", get "recipes" "/api/v1/workspaces/acme/context-recipes" ]
    )


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        GotApi raw -> handleResponse raw model
        SelectIncident incident -> ( { model | active = IncidentActive incident, posts = [], coordination = emptyCoordination, collections = [], similar = [], summaryDraft = incident.verifiedSummary, error = Nothing }, Cmd.batch [ get "posts" (incidentPath model incident.id ++ "/posts"), get "coordination" (incidentPath model incident.id ++ "/coordination"), get "collections" (incidentPath model incident.id ++ "/context-collections"), getAs "similar" (incidentPath model incident.id ++ "/similar") model.actor ] )
        SelectChannel channel -> ( { model | active = ChannelActive channel, posts = [], error = Nothing }, get "posts" (channelPath model channel.id ++ "/posts") )
        SetDraft v -> ( { model | draft = v }, Cmd.none )
        SetBlockType v -> ( { model | blockType = v }, Cmd.none )
        SetReply post -> ( { model | replyTo = Just post, editing = Nothing }, Cmd.none )
        EditPost post ->
            case post.blocks of
                first :: _ -> ( { model | editing = Just post, replyTo = Nothing, draft = first.body, blockType = first.kind }, Cmd.none )
                [] -> ( model, Cmd.none )
        CancelReply -> ( { model | replyTo = Nothing, editing = Nothing, draft = "" }, Cmd.none )
        SetCreateTitle v -> ( { model | createTitle = v }, Cmd.none )
        SetCreateSeverity v -> ( { model | createSeverity = v }, Cmd.none )
        SetCreateScope v -> ( { model | createScope = v }, Cmd.none )
        SetTemplate v -> ( { model | templateId = v }, Cmd.none )
        SetSummary v -> ( { model | summaryDraft = v }, Cmd.none )
        SetActor v -> ( { model | actor = v }, Cmd.none )
        SetMember v -> ( { model | memberDraft = v }, Cmd.none )
        SetMemberRole v -> ( { model | memberRole = v }, Cmd.none )
        SetStructured v -> ( { model | structuredDraft = v }, Cmd.none )
        SetStructuredType v -> ( { model | structuredType = v }, Cmd.none )
        SetRecipe v -> ( { model | recipeId = v }, Cmd.none )
        CollectContext -> incidentCommand model "POST" "/context-collections" (Encode.object [ ( "recipeId", Encode.string model.recipeId ), ( "labels", Encode.object [] ) ])
        RefreshContext collection -> incidentCommand model "POST" ("/context-collections/" ++ collection.id ++ "/refresh") (Encode.object [ ( "labels", Encode.object [] ) ])
        DismissError -> ( { model | error = Nothing }, Cmd.none )
        Reload -> ( { model | busy = True }, Cmd.batch [ get "incidents" (workspacePath model ++ "/incidents"), get "channels" (workspacePath model ++ "/permanent-channels") ] )
        CreateIncident ->
            if String.trim model.createTitle == "" then ( model, Cmd.none )
            else
                let
                    fields = [ ( "title", Encode.string (String.trim model.createTitle) ) ]
                        ++ (if model.templateId == "" then [ ( "severity", Encode.string model.createSeverity ), ( "scope", Encode.list Encode.string (splitScope model.createScope) ) ] else [ ( "templateId", Encode.string model.templateId ) ])
                in command model "mutate" "POST" (workspacePath model ++ "/incidents") (Encode.object fields)
        CreateChannel ->
            if String.trim model.createTitle == "" then ( model, Cmd.none )
            else command model "mutate" "POST" (workspacePath model ++ "/permanent-channels") (Encode.object [ ( "title", Encode.string (String.trim model.createTitle) ) ])
        Publish -> publish model
        AdvanceLifecycle ->
            case model.active of
                IncidentActive incident -> command model "incident" "PATCH" (incidentPath model incident.id) (Encode.object [ ( "lifecycle", Encode.string (nextLifecycle incident.lifecycle) ) ])
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
        TransferOwnership ->
            case model.active of
                IncidentActive incident -> command model "incident" "POST" (incidentPath model incident.id ++ "/ownership-transfers") (Encode.object [ ( "newOwnerId", Encode.string (String.trim model.memberDraft) ) ])
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
                Just post -> [ ( "replyToPostId", Encode.string post.id ) ]
                Nothing -> []
            block =
                Encode.object
                    [ ( "type", Encode.string model.blockType )
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
        if path == "" then ( model, Cmd.none ) else command { model | draft = "", replyTo = Nothing, editing = Nothing } "post" method target payload


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
                "collections" -> case Decode.decodeValue (Decode.field "items" (Decode.list contextCollectionDecoder)) response.body of
                    Ok items -> ( { model | collections = items, busy = False }, Cmd.none )
                    Err err -> decodeFailure model err
                "similar" -> case Decode.decodeValue (Decode.field "items" (Decode.list similarIncidentDecoder)) response.body of
                    Ok items -> ( { model | similar = items, busy = False }, Cmd.none )
                    Err err -> decodeFailure model err
                "posts" -> case Decode.decodeValue (Decode.field "items" (Decode.list postDecoder)) response.body of
                    Ok items -> ( { model | posts = items, busy = False }, Cmd.none )
                    Err err -> decodeFailure model err
                "coordination" -> case Decode.decodeValue coordinationDecoder response.body of
                    Ok items -> ( { model | coordination = items, busy = False }, Cmd.none )
                    Err err -> decodeFailure model err
                "incident" -> case Decode.decodeValue incidentDecoder response.body of
                    Ok incident -> ( replaceIncident incident { model | active = IncidentActive incident, summaryDraft = incident.verifiedSummary, busy = False }, Cmd.none )
                    Err err -> decodeFailure model err
                "post" -> ( { model | busy = False }, refreshPosts model )
                _ -> ( { model | createTitle = "", memberDraft = "", busy = False }, Cmd.batch [ get "incidents" (workspacePath model ++ "/incidents"), get "channels" (workspacePath model ++ "/permanent-channels"), refreshPosts model, refreshCoordination model, refreshCollections model ] )


replaceIncident : Incident -> Model -> Model
replaceIncident incident model = { model | incidents = List.map (\item -> if item.id == incident.id then incident else item) model.incidents }

decodeFailure model err = ( { model | busy = False, error = Just (Decode.errorToString err) }, Cmd.none )
errorMessage response = Decode.decodeValue (Decode.field "message" Decode.string) response.body |> Result.withDefault ("Request failed with status " ++ String.fromInt response.status)


view : Model -> Html Msg
view model =
    div [ class "app-shell" ]
        [ header [ class "top-bar" ] [ h1 [] [ text "Gimme Context" ], span [ class "workspace-name" ] [ text model.workspace ], label [ class "actor" ] [ text "Acting as", input [ value model.actor, onInput SetActor, attribute "aria-label" "Current principal" ] [] ] ]
        , case model.error of
            Just message ->
                div [ class "error-banner", attribute "role" "alert" ] [ text message, button [ onClick DismissError ] [ text "Dismiss" ] ]

            Nothing ->
                text ""
        , div [ class "workspace" ] [ navigation model, main_ [ class "incident" ] [ content model ] ]
        ]


navigation model =
    aside [ class "channel-nav", attribute "aria-label" "Channels" ]
        [ p [ class "eyebrow" ] [ text "INCIDENTS" ], ul [] (List.map (incidentNav model) model.incidents)
        , p [ class "eyebrow" ] [ text "PERMANENT" ], ul [] (List.map (channelNav model) model.channels)
        , label [] [ text "New channel title", input [ value model.createTitle, placeholder "Checkout incident", onInput SetCreateTitle ] [] ]
        , label [] [ text "Incident severity", select [ value model.createSeverity, onChange SetCreateSeverity ] (List.map (choice model.createSeverity) [ "unclassified", "SEV-1", "SEV-2", "SEV-3", "SEV-4" ]) ]
        , label [] [ text "Incident scope", input [ value model.createScope, placeholder "checkout, production", onInput SetCreateScope ] [] ]
        , label [] [ text "Incident template", select [ value model.templateId, onChange SetTemplate ] (option [ value "" ] [ text "Workspace defaults" ] :: List.map (templateChoice model.templateId) model.templates) ]
        , div [ class "nav-actions" ] [ button [ onClick CreateIncident, disabled model.busy ] [ text "New incident" ], button [ onClick CreateChannel, disabled model.busy ] [ text "New permanent" ] ]
        ]

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
incidentHeader model incident = pageHeader incident.title (incident.severity ++ " · " ++ incident.lifecycle) (Just (button [ class "primary-action", onClick AdvanceLifecycle, disabled (model.busy || nextLifecycle incident.lifecycle == incident.lifecycle) ] [ text (if incident.lifecycle == "resolved" then "Resolved" else "Advance to " ++ nextLifecycle incident.lifecycle) ]))


feed model = section [ class "feed", attribute "aria-label" "Chronological feed" ] (List.map (viewPost model) model.posts ++ [ composer model ])
viewPost model post =
    Html.article [ class "post" ]
        [ div [ class "post-meta" ] [ span [ class "author" ] [ text post.authorId ], span [] [ text ("revision " ++ String.fromInt post.revision) ], if post.replyToPostId /= "" then span [] [ text "reply" ] else text "" ]
        , div [] (List.map viewBlock post.blocks)
        , div [] [ button [ class "text-action", onClick (SetReply post) ] [ text "Reply" ], if post.authorId == model.actor then button [ class "text-action edit-action", onClick (EditPost post) ] [ text "Edit" ] else text "" ]
        ]
viewBlock block = div [ class ("block block-" ++ block.kind) ] [ span [ class "block-kind" ] [ text block.kind ], p [] [ text block.body ] ]

composer model =
    div [ class "composer" ]
        [ case model.editing of
            Just post ->
                div [ class "replying" ] [ text ("Editing revision " ++ String.fromInt post.revision), button [ onClick CancelReply ] [ text "Cancel" ] ]

            Nothing ->
                case model.replyTo of
                    Just post ->
                        div [ class "replying" ] [ text ("Replying to " ++ post.authorId), button [ onClick CancelReply ] [ text "Cancel" ] ]

                    Nothing ->
                        text ""
        , label [] [ text "Add to the channel" ], textarea [ placeholder "Share an update…", value model.draft, onInput SetDraft ] []
        , div [ class "composer-actions" ] [ select [ value model.blockType, onChange SetBlockType, attribute "aria-label" "Block type" ] (List.map (choice model.blockType) [ "markdown", "code", "log", "table", "checklist", "fact", "decision", "action", "poll", "approval", "status" ]), button [ class "primary-action", onClick Publish, disabled model.busy ] [ text "Post update" ] ]
        ]

incidentState model incident =
    aside [ class "state-panel" ]
        [ h2 [] [ text "Incident state" ], stateField "Owner" incident.ownerId, stateField "Scope" (String.join ", " incident.scope)
        , h2 [] [ text "Verified summary" ], textarea [ value model.summaryDraft, onInput SetSummary, placeholder "Outcome and verified impact" ] [], button [ onClick SaveSummary, disabled model.busy ] [ text "Save summary" ]
        , h2 [] [ text "Closure checklist" ], div [] (List.map (checkItem model) incident.closureChecklist)
        , h2 [] [ text "Participants and ownership" ], input [ value model.memberDraft, onInput SetMember, placeholder "Principal ID", attribute "aria-label" "New participant" ] [], select [ value model.memberRole, onChange SetMemberRole, attribute "aria-label" "Participant role" ] (List.map (choice model.memberRole) [ "participant", "editor", "viewer" ]), button [ onClick AddMember, disabled model.busy ] [ text "Add participant" ], button [ onClick TransferOwnership, disabled model.busy ] [ text "Transfer ownership" ]
        , h2 [] [ text "Structured coordination" ], select [ value model.structuredType, onChange SetStructuredType, attribute "aria-label" "Coordination type" ] (List.map (choice model.structuredType) [ "fact", "decision", "action", "poll" ]), textarea [ value model.structuredDraft, onInput SetStructured, placeholder "Statement, task, or question" ] [], button [ onClick AddStructured, disabled model.busy ] [ text "Add" ]
        , contextPanel model
        , coordinationView model
        ]

contextPanel model =
    div [ class "context-panel" ]
        [ h2 [] [ text "Operational context" ]
        , select [ value model.recipeId, onChange SetRecipe, attribute "aria-label" "Context recipe" ] (option [ value "" ] [ text "Select recipe" ] :: List.map contextRecipeChoice model.recipes)
        , button [ onClick CollectContext, disabled (model.busy || model.recipeId == "") ] [ text "Collect context" ]
        , div [] (List.map viewCollection model.collections)
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
        IncidentActive i -> get "posts" (incidentPath model i.id ++ "/posts")
        ChannelActive c -> get "posts" (channelPath model c.id ++ "/posts")
        None -> Cmd.none
refreshCoordination model =
    case model.active of
        IncidentActive i -> get "coordination" (incidentPath model i.id ++ "/coordination")
        _ -> Cmd.none
refreshCollections model =
    case model.active of
        IncidentActive i -> get "collections" (incidentPath model i.id ++ "/context-collections")
        _ -> Cmd.none

responseDecoder = Decode.map4 Response (Decode.field "tag" Decode.string) (Decode.field "ok" Decode.bool) (Decode.field "status" Decode.int) (Decode.field "body" Decode.value)
incidentDecoder = Decode.map8 Incident (Decode.field "id" Decode.string) (Decode.field "title" Decode.string) (Decode.field "ownerId" Decode.string) (Decode.field "severity" Decode.string) (Decode.field "lifecycle" Decode.string) (Decode.field "scope" (Decode.list Decode.string)) (Decode.oneOf [ Decode.field "verifiedSummary" Decode.string, Decode.succeed "" ]) (Decode.field "closureChecklist" (Decode.list checklistDecoder))
checklistDecoder = Decode.map3 ChecklistItem (Decode.field "id" Decode.string) (Decode.field "label" Decode.string) (Decode.field "completed" Decode.bool)
channelDecoder = Decode.map3 Channel (Decode.field "id" Decode.string) (Decode.field "title" Decode.string) (Decode.oneOf [ Decode.field "description" Decode.string, Decode.succeed "" ])
templateDecoder = Decode.map3 Template (Decode.field "id" Decode.string) (Decode.field "name" Decode.string) (Decode.field "version" Decode.int)
postDecoder = Decode.map6 Post (Decode.field "id" Decode.string) (Decode.field "authorId" Decode.string) (Decode.field "revision" Decode.int) (Decode.oneOf [ Decode.field "replyToPostId" Decode.string, Decode.succeed "" ]) (Decode.field "blocks" (Decode.list blockDecoder)) (Decode.field "createdAt" Decode.string)
blockDecoder = Decode.map3 Block (Decode.field "id" Decode.string) (Decode.field "type" Decode.string) (Decode.oneOf [ Decode.at [ "payload", "text" ] Decode.string, Decode.at [ "payload", "label" ] Decode.string, Decode.succeed "Structured block" ])
coordinationDecoder = Decode.map5 Coordination (Decode.field "facts" (Decode.list factDecoder)) (Decode.field "decisions" (Decode.list decisionDecoder)) (Decode.field "actions" (Decode.list actionDecoder)) (Decode.field "polls" (Decode.list pollDecoder)) (Decode.field "approvals" (Decode.list approvalDecoder))
factDecoder = Decode.map3 Fact (Decode.field "id" Decode.string) (Decode.field "statement" Decode.string) (Decode.field "state" Decode.string)
decisionDecoder = Decode.map3 Decision (Decode.field "id" Decode.string) (Decode.field "statement" Decode.string) (Decode.field "status" Decode.string)
actionDecoder = Decode.map3 Action (Decode.field "id" Decode.string) (Decode.field "title" Decode.string) (Decode.field "status" Decode.string)
pollDecoder = Decode.map3 Poll (Decode.field "id" Decode.string) (Decode.field "question" Decode.string) (Decode.field "options" (Decode.list pollOptionDecoder))
pollOptionDecoder = Decode.map2 PollOption (Decode.field "id" Decode.string) (Decode.field "label" Decode.string)
approvalDecoder = Decode.map3 Approval (Decode.field "id" Decode.string) (Decode.field "actionId" Decode.string) (Decode.field "outcome" Decode.string)
contextRecipeDecoder = Decode.map3 ContextRecipe (Decode.field "id" Decode.string) (Decode.field "name" Decode.string) (Decode.field "version" Decode.int)
contextCollectionDecoder = Decode.map4 ContextCollection (Decode.field "id" Decode.string) (Decode.field "status" Decode.string) (Decode.field "snapshots" (Decode.list contextSnapshotDecoder)) (Decode.field "failures" (Decode.list retrievalFailureDecoder))
contextSnapshotDecoder = Decode.map3 ContextSnapshot (Decode.field "source" Decode.string) (Decode.field "query" Decode.string) (Decode.field "retrievedAt" Decode.string)
retrievalFailureDecoder = Decode.map4 RetrievalFailure (Decode.field "source" Decode.string) (Decode.field "category" Decode.string) (Decode.field "message" Decode.string) (Decode.field "requiredHumanAction" Decode.string)
similarIncidentDecoder = Decode.map2 SimilarIncident (Decode.field "incident" incidentDecoder) (Decode.field "score" Decode.int)

subscriptions _ = apiResponse GotApi
main = Browser.element { init = init, update = update, subscriptions = subscriptions, view = view }
