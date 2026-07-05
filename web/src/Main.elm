module Main exposing (main)

import Browser
import Html exposing (Html, aside, button, div, h1, h2, header, input, label, li, main_, p, section, span, text, textarea, ul)
import Html.Attributes exposing (class, placeholder, value)
import Html.Events exposing (onClick, onInput)


type alias Model =
    { lifecycle : String
    , draft : String
    , posts : List Post
    }


type alias Post =
    { author : String, kind : String, body : String, time : String }


type Msg
    = SetDraft String
    | Publish
    | AdvanceLifecycle


init : Model
init =
    { lifecycle = "Investigating"
    , draft = ""
    , posts =
        [ { author = "Alert routing", kind = "Status", body = "Incident opened manually. Coordination remains available without integrations or AI.", time = "09:42" }
        , { author = "A. Tan", kind = "Fact", body = "Checkout error rate increased after the 09:35 deployment.", time = "09:47" }
        ]
    }


update : Msg -> Model -> Model
update msg model =
    case msg of
        SetDraft value_ ->
            { model | draft = value_ }

        Publish ->
            if String.trim model.draft == "" then
                model

            else
                { model
                    | draft = ""
                    , posts = model.posts ++ [ { author = "You", kind = "Markdown", body = String.trim model.draft, time = "Now" } ]
                }

        AdvanceLifecycle ->
            { model | lifecycle = nextLifecycle model.lifecycle }


nextLifecycle : String -> String
nextLifecycle current =
    case current of
        "Investigating" -> "Mitigating"
        "Mitigating" -> "Monitoring"
        "Monitoring" -> "Resolved"
        _ -> current


view : Model -> Html Msg
view model =
    div [ class "app-shell" ]
        [ header [ class "top-bar" ]
            [ h1 [] [ text "Gimme Context" ]
            , span [ class "workspace-name" ] [ text "Acme Operations" ]
            ]
        , div [ class "workspace" ]
            [ aside [ class "channel-nav" ]
                [ p [ class "eyebrow" ] [ text "INCIDENTS" ]
                , ul []
                    [ li [ class "selected-channel" ] [ text "SEV-2 Checkout errors" ]
                    , li [] [ text "SEV-3 Queue delay" ]
                    ]
                , p [ class "eyebrow" ] [ text "PERMANENT" ]
                , ul [] [ li [] [ text "Platform" ], li [] [ text "Payments" ] ]
                ]
            , main_ [ class "incident" ]
                [ incidentHeader model
                , div [ class "incident-layout" ]
                    [ section [ class "feed", Html.Attributes.attribute "aria-label" "Incident feed" ]
                        (List.map viewPost model.posts ++ [ composer model ])
                    , incidentState model
                    ]
                ]
            ]
        ]


incidentHeader : Model -> Html Msg
incidentHeader model =
    header [ class "incident-header" ]
        [ div []
            [ div [ class "incident-kicker" ] [ span [ class "severity" ] [ text "SEV-2" ], span [] [ text model.lifecycle ] ]
            , h2 [] [ text "Checkout errors above threshold" ]
            , p [] [ text "Customer checkout · production · started 09:35 SGT" ]
            ]
        , button [ class "primary-action", onClick AdvanceLifecycle ] [ text (advanceLabel model.lifecycle) ]
        ]


advanceLabel : String -> String
advanceLabel lifecycle =
    if lifecycle == "Resolved" then "Resolved" else "Advance to " ++ nextLifecycle lifecycle


viewPost : Post -> Html Msg
viewPost post =
    article_ [ class "post" ]
        [ div [ class "post-meta" ] [ span [ class "author" ] [ text post.author ], span [] [ text post.time ] ]
        , div [ class ("block block-" ++ String.toLower post.kind) ]
            [ span [ class "block-kind" ] [ text post.kind ]
            , p [] [ text post.body ]
            , button [ class "text-action" ] [ text "Reply" ]
            ]
        ]


article_ : List (Html.Attribute msg) -> List (Html msg) -> Html msg
article_ =
    Html.article


composer : Model -> Html Msg
composer model =
    div [ class "composer" ]
        [ label [] [ text "Add to the incident" ]
        , textarea [ placeholder "Share an update, fact, decision, or action…", value model.draft, onInput SetDraft ] []
        , div [ class "composer-actions" ]
            [ button [ class "type-selector" ] [ text "Markdown ▾" ]
            , button [ class "primary-action", onClick Publish ] [ text "Post update" ]
            ]
        ]


incidentState : Model -> Html Msg
incidentState model =
    aside [ class "state-panel" ]
        [ h2 [] [ text "Incident state" ]
        , stateField "Lifecycle" model.lifecycle
        , stateField "Owner" "A. Tan"
        , stateField "Scope" "Checkout, production"
        , stateField "Last update" "Just now"
        , h2 [] [ text "Closure" ]
        , checklist "Impact understood" True
        , checklist "Mitigation verified" False
        , checklist "Follow-ups assigned" False
        ]


stateField : String -> String -> Html Msg
stateField name content =
    div [ class "state-field" ] [ span [] [ text name ], strong_ [] [ text content ] ]


strong_ : List (Html.Attribute msg) -> List (Html msg) -> Html msg
strong_ =
    Html.strong


checklist : String -> Bool -> Html Msg
checklist content checked =
    label [ class "check-item" ]
        [ input [ Html.Attributes.type_ "checkbox", Html.Attributes.checked checked ] []
        , text content
        ]


main : Program () Model Msg
main =
    Browser.sandbox { init = init, update = update, view = view }
