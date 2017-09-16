module Main exposing (main)

import Data.Session as Session exposing (Session)
import Data.User as User exposing (User, Username)
import Html exposing (..)
import Json.Decode as Decode exposing (Value)
import Navigation exposing (Location)
import Page.Errored as Errored exposing (PageLoadError)
import Page.Home as Home
import Page.Login as Login
import Page.NotFound as NotFound
import Page.Projects as Projects
import Page.Project as Project
import Ports
import Route exposing (Route)
import Task
import Util exposing ((=>))
import Views.Page as Page exposing (ActivePage)


type Page
    = Blank
    | NotFound
    | Errored PageLoadError
    | Home Home.Model
    | Projects Projects.Model
    | Project Project.Model
    | Login Login.Model


type PageState
    = Loaded Page
    | TransitioningFrom Page



-- MODEL --


type alias Model =
    { session : Session
    , pageState : PageState
    }


init : Value -> Location -> ( Model, Cmd Msg )
init val location =
    setRoute (Route.fromLocation location)
        { pageState = Loaded initialPage
        , session = { user = decodeUserFromJson val }
        }


decodeUserFromJson : Value -> Maybe User
decodeUserFromJson json =
    json
        |> Decode.decodeValue Decode.string
        |> Result.toMaybe
        |> Maybe.andThen (Decode.decodeString User.decoder >> Result.toMaybe)


initialPage : Page
initialPage =
    Blank



-- VIEW --


view : Model -> Html Msg
view model =
    case model.pageState of
        Loaded page ->
            viewPage model.session False page

        TransitioningFrom page ->
            viewPage model.session True page


viewPage : Session -> Bool -> Page -> Html Msg
viewPage session isLoading page =
    let
        frame =
            Page.frame isLoading session.user
    in
        case page of
            NotFound ->
                NotFound.view session
                    |> frame Page.Other

            Blank ->
                -- This is for the very initial page load, while we are loading
                -- data via HTTP. We could also render a spinner here.
                Html.text ""
                    |> frame Page.Other

            Errored subModel ->
                Errored.view session subModel
                    |> frame Page.Other

            Home subModel ->
                Home.view session subModel
                    |> frame Page.Home
                    |> Html.map HomeMsg

            Projects subModel ->
                Projects.view session subModel
                    |> frame Page.Projects
                    |> Html.map ProjectsMsg

            Project subModel ->
                Project.view session subModel
                    |> frame Page.Project
                    |> Html.map ProjectMsg

            Login subModel ->
                Login.view session subModel
                    |> frame Page.Login
                    |> Html.map LoginMsg



-- SUBSCRIPTIONS --


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.map SetUser sessionChange


sessionChange : Sub (Maybe User)
sessionChange =
    Ports.onSessionChange (Decode.decodeValue User.decoder >> Result.toMaybe)


getPage : PageState -> Page
getPage pageState =
    case pageState of
        Loaded page ->
            page

        TransitioningFrom page ->
            page



-- UPDATE --


type Msg
    = SetRoute (Maybe Route)
    | HomeMsg Home.Msg
    | HomeLoaded (Result PageLoadError Home.Model)
    | SetUser (Maybe User)
    | LoginMsg Login.Msg
    | ProjectsLoaded (Result PageLoadError Projects.Model)
    | ProjectsMsg Projects.Msg
    | ProjectLoaded (Result PageLoadError Project.Model)
    | ProjectMsg Project.Msg


setRoute : Maybe Route -> Model -> ( Model, Cmd Msg )
setRoute maybeRoute model =
    let
        transition toMsg task =
            { model | pageState = TransitioningFrom (getPage model.pageState) }
                => Task.attempt toMsg task

        errored =
            pageErrored model
    in
        case maybeRoute of
            Nothing ->
                { model | pageState = Loaded NotFound } => Cmd.none

            Just (Route.Home) ->
                case model.session.user of
                    Just user ->
                        transition HomeLoaded (Home.init model.session)

                    Nothing ->
                        model => Route.modifyUrl Route.Login

            Just (Route.Login) ->
                { model | pageState = Loaded (Login Login.initialModel) } => Cmd.none

            Just (Route.Projects) ->
                case model.session.user of
                    Just user ->
                        transition ProjectsLoaded (Projects.init model.session)

                    Nothing ->
                        errored Page.Projects "You must be signed in to access your projects."

            Just (Route.Project id) ->
                case model.session.user of
                    Just user ->
                        transition ProjectLoaded (Project.init model.session id)

                    Nothing ->
                        errored Page.Project "You must be signed in to access this project."


pageErrored : Model -> ActivePage -> String -> ( Model, Cmd msg )
pageErrored model activePage errorMessage =
    let
        error =
            Errored.pageLoadError activePage errorMessage
    in
        { model | pageState = Loaded (Errored error) } => Cmd.none


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    updatePage (getPage model.pageState) msg model


updatePage : Page -> Msg -> Model -> ( Model, Cmd Msg )
updatePage page msg model =
    let
        session =
            model.session

        toPage toModel toMsg subUpdate subMsg subModel =
            let
                ( newModel, newCmd ) =
                    subUpdate subMsg subModel
            in
                ( { model | pageState = Loaded (toModel newModel) }, Cmd.map toMsg newCmd )

        errored =
            pageErrored model
    in
        case ( msg, page ) of
            ( SetRoute route, _ ) ->
                setRoute route model

            ( SetUser user, _ ) ->
                let
                    session =
                        model.session

                    cmd =
                        -- If we just signed out, then redirect to Home.
                        if session.user /= Nothing && user == Nothing then
                            Route.modifyUrl Route.Home
                        else
                            Cmd.none
                in
                    { model | session = { session | user = user } }
                        => cmd

            ( LoginMsg subMsg, Login subModel ) ->
                let
                    ( ( pageModel, cmd ), msgFromPage ) =
                        Login.update subMsg subModel

                    newModel =
                        case msgFromPage of
                            Login.NoOp ->
                                model

                            Login.SetUser user ->
                                let
                                    session =
                                        model.session
                                in
                                    { model | session = { user = Just user } }
                in
                    { newModel | pageState = Loaded (Login pageModel) }
                        => Cmd.map LoginMsg cmd

            ( HomeLoaded (Ok subModel), _ ) ->
                { model | pageState = Loaded (Home subModel) } => Cmd.none

            ( HomeLoaded (Err error), _ ) ->
                { model | pageState = Loaded (Errored error) } => Cmd.none

            ( HomeMsg subMsg, Home subModel ) ->
                toPage Home HomeMsg (Home.update session) subMsg subModel

            ( ProjectsLoaded (Ok subModel), _ ) ->
                { model | pageState = Loaded (Projects subModel) } => Cmd.none

            ( ProjectsLoaded (Err error), _ ) ->
                { model | pageState = Loaded (Errored error) } => Cmd.none

            ( ProjectsMsg subMsg, Projects subModel ) ->
                toPage Projects ProjectsMsg (Projects.update session) subMsg subModel

            ( ProjectLoaded (Ok subModel), _ ) ->
                { model | pageState = Loaded (Project subModel) } => Cmd.none

            ( ProjectLoaded (Err error), _ ) ->
                { model | pageState = Loaded (Errored error) } => Cmd.none

            ( ProjectMsg subMsg, Project subModel ) ->
                toPage Project ProjectMsg (Project.update session) subMsg subModel

            ( _, NotFound ) ->
                -- Disregard incoming messages when we're on the
                -- NotFound page.
                model => Cmd.none

            ( _, _ ) ->
                -- Disregard incoming messages that arrived for the wrong page
                model => Cmd.none



-- MAIN --


main : Program Value Model Msg
main =
    Navigation.programWithFlags (Route.fromLocation >> SetRoute)
        { init = init
        , view = view
        , update = update
        , subscriptions = subscriptions
        }