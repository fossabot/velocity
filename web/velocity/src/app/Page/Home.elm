module Page.Home exposing (view, update, Model, Msg, init)

{-| The homepage. You can get here via either the / or /#/ routes.
-}

import Html exposing (..)
import Html.Attributes exposing (class, href, id, placeholder, attribute, classList, style)
import Data.Session as Session exposing (Session)
import Data.Project as Project exposing (Project)
import Page.Errored as Errored exposing (PageLoadError, pageLoadError)
import Data.PaginatedList as PaginatedList exposing (Paginated(..))
import Util exposing ((=>), onClickStopPropagation)
import Views.Page as Page
import Task exposing (Task)
import Http
import Request.Project
import Page.Helpers exposing (formatDate, sortByDatetime)
import Route
import Page.Project.Route as ProjectRoute
import Time.DateTime as DateTime
import Views.Helpers exposing (onClickPage)
import Navigation exposing (newUrl)


-- MODEL --


type alias Model =
    { projects : List Project }


init : Session -> Task PageLoadError Model
init session =
    let
        maybeAuthToken =
            Maybe.map .token session.user

        loadProjects =
            Request.Project.list maybeAuthToken
                |> Http.toTask

        handleLoadError _ =
            pageLoadError Page.Home "Homepage is currently unavailable."
    in
        Task.map (\(Paginated { results }) -> Model results) loadProjects
            |> Task.mapError handleLoadError


view : Session -> Model -> Html Msg
view session model =
    div [ class "container-fluid" ]
        [ div [ class "row default-margin-top" ]
            [ div [ class "col-12 col-md-6" ]
                [ div [ class "card" ]
                    [ h4 [ class "card-header" ]
                        [ text "Last builds" ]
                    , ul [ class "list-group" ]
                        [ li [ class "list-group-item list-group-item-action flex-column align-items-start" ]
                            [ div [ class "d-flex w-100 justify-content-between" ]
                                [ h5 [ class "mb-1" ]
                                    [ text "List group item heading" ]
                                , small []
                                    [ text "3 days ago" ]
                                ]
                            , p [ class "mb-1" ]
                                [ text "Donec id elit non mi porta gravida at eget metus. Maecenas sed diam eget risus varius blandit." ]
                            , small []
                                [ text "Donec id elit non mi porta." ]
                            ]
                        , li [ class "list-group-item list-group-item-action flex-column align-items-start" ]
                            [ div [ class "d-flex w-100 justify-content-between" ]
                                [ h5 [ class "mb-1" ]
                                    [ text "List group item heading" ]
                                , small [ class "text-muted" ]
                                    [ text "3 days ago" ]
                                ]
                            ]
                        , li [ class "list-group-item list-group-item-action flex-column align-items-start" ]
                            [ div [ class "d-flex w-100 justify-content-between" ]
                                [ h5 [ class "mb-1" ]
                                    [ text "List group item heading" ]
                                , small [ class "text-muted" ]
                                    [ text "3 days ago" ]
                                ]
                            ]
                        ]
                    ]
                ]
            , div [ class "col-12 col-md-6" ]
                [ div [ class "card" ]
                    [ h4
                        [ class "card-header" ]
                        [ a
                            [ Route.href Route.Projects
                            , onClickPage NewUrl Route.Projects
                            ]
                            [ text "Projects" ]
                        ]
                    , viewProjectList model.projects
                    ]
                ]
            ]
        ]


viewProjectList : List Project -> Html Msg
viewProjectList projects =
    let
        latestProjects =
            sortByDatetime .updatedAt projects
    in
        ul [ class "list-group" ] (List.map viewProjectListItem latestProjects)


viewProjectListItem : Project -> Html Msg
viewProjectListItem project =
    let
        route =
            Route.Project project.id ProjectRoute.Overview
    in
        li [ class "list-group-item list-group-item-action flex-column align-items-start" ]
            [ div [ class "d-flex w-100 justify-content-between" ]
                [ h5 [ class "mb-1" ]
                    [ a
                        [ Route.href route
                        , onClickPage NewUrl route
                        ]
                        [ text project.name ]
                    ]
                , small []
                    [ DateTime.date project.updatedAt |> formatDate |> text ]
                ]
            , small []
                [ text project.repository ]
            ]



-- UPDATE --


type Msg
    = NewUrl String


update : Session -> Msg -> Model -> ( Model, Cmd Msg )
update session msg model =
    case msg of
        NewUrl url ->
            model => newUrl url
