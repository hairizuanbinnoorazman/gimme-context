module Main exposing (main)

import Browser
import Html exposing (Html, div, h1, header, main_, p, text)
import Html.Attributes exposing (class)
import View.Components as Components


type alias Model =
    { apiState : ApiState }


type ApiState
    = NotChecked
    | Available


type Msg
    = CheckAPI


init : Model
init =
    { apiState = NotChecked }


update : Msg -> Model -> Model
update msg model =
    case msg of
        CheckAPI ->
            { model | apiState = Available }


view : Model -> Html Msg
view model =
    div [ class "app-shell" ]
        [ header [ class "top-bar" ]
            [ h1 [] [ text "Gimme Context" ] ]
        , main_ [ class "main-content" ]
            [ Components.surface
                [ h1 [] [ text "Incident coordination, with the relevant context." ]
                , p [] [ text (statusText model.apiState) ]
                , Components.primaryButton CheckAPI "Check local shell"
                ]
            ]
        ]


statusText : ApiState -> String
statusText state =
    case state of
        NotChecked ->
            "The Phase 1 application shell is ready."

        Available ->
            "The Elm update loop is working."


main : Program () Model Msg
main =
    Browser.sandbox
        { init = init
        , update = update
        , view = view
        }
