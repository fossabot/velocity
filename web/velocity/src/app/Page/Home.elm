module Page.Home exposing (view, update, Model, Msg, ExternalMsg(..), init, channelName, initialEvents, subscriptions)

{-| The homepage. You can get here via either the / or /#/ routes.
-}

-- EXTERNAL

import Http
import Html exposing (..)
import Html.Attributes exposing (class, href, id, placeholder, attribute, classList, style)
import Html.Events exposing (onClick)
import Task exposing (Task)
import Navigation exposing (newUrl)
import Dict exposing (Dict)
import Time.DateTime as DateTime
import Json.Encode as Encode
import Json.Decode as Decode
import Bootstrap.Modal as Modal
import Bootstrap.Button as Button
import Json.Decode as Decode exposing (decodeString)


-- INTERNAL

import Context exposing (Context)
import Component.ProjectForm as ProjectForm
import Component.KnownHostForm as KnownHostForm
import Component.Form as Form
import Data.Session as Session exposing (Session)
import Data.Project as Project exposing (Project, addProject)
import Data.KnownHost as KnownHost exposing (KnownHost, addKnownHost)
import Data.PaginatedList as PaginatedList exposing (Paginated(..))
import Data.GitUrl as GitUrl exposing (GitUrl)
import Page.Errored as Errored exposing (PageLoadError, pageLoadError)
import Page.Helpers exposing (formatDate, sortByDatetime)
import Page.Project.Route as ProjectRoute
import Views.Helpers exposing (onClickPage)
import Views.Page as Page
import Util exposing ((=>), onClickStopPropagation, viewIf)
import Request.Project
import Request.KnownHost
import Request.Errors
import Route
import Dom
import Ports


-- MODEL --


type alias Model =
    { projects : List Project
    , knownHosts : List KnownHost
    , newProjectForm : ProjectForm.Context
    , newKnownHostForm : KnownHostForm.Context
    , newProjectModalVisibility : Modal.Visibility
    }


init : Context -> Session msg -> Task (Request.Errors.Error PageLoadError) Model
init context session =
    let
        maybeAuthToken =
            Maybe.map .token session.user

        loadProjects =
            Request.Project.list context maybeAuthToken

        loadKnownHosts =
            Request.KnownHost.list context maybeAuthToken

        errorPage =
            pageLoadError Page.Home "Homepage is currently unavailable."

        initialModel (Paginated projects) (Paginated knownHosts) =
            { projects = projects.results
            , knownHosts = knownHosts.results
            , newProjectForm = ProjectForm.init
            , newKnownHostForm = KnownHostForm.init
            , newProjectModalVisibility = Modal.hidden
            }
    in
        Task.map2 initialModel loadProjects loadKnownHosts
            |> Task.mapError (Request.Errors.withDefaultError errorPage)



-- SUBSCRIPTIONS --


subscriptions : Model -> Sub Msg
subscriptions { newProjectModalVisibility } =
    let
        modalSubs =
            Modal.subscriptions newProjectModalVisibility AnimateNewProjectModal

        gitUrlSub =
            Sub.map SetGitUrl gitUrlParsed
    in
        Sub.batch
            [ modalSubs, gitUrlSub ]


gitUrlParsed : Sub (Maybe GitUrl)
gitUrlParsed =
    Ports.onGitUrlParsed (Decode.decodeValue GitUrl.decoder >> Result.toMaybe)



-- CHANNELS --


channelName : String
channelName =
    "projects"


initialEvents : Dict String (List ( String, Encode.Value -> Msg ))
initialEvents =
    let
        pageEvents =
            [ ( "project:new", AddProject ) ]
    in
        Dict.singleton channelName pageEvents



-- VIEW --


view : Session msg -> Model -> Html Msg
view session model =
    let
        hasProjects =
            not (List.isEmpty model.projects)

        projectList =
            viewProjectList model.projects
    in
        div [ class "py-2 my-4" ]
            [ viewToolbar
            , viewIf hasProjects projectList
            , viewNewProjectModal model
            ]


viewToolbar : Html Msg
viewToolbar =
    div [ class "btn-toolbar d-flex flex-row-reverse" ]
        [ button
            [ class "btn btn-primary btn-lg"
            , style [ "border-radius" => "25px" ]
            , onClick ShowNewProjectModal
            ]
            [ i [ class "fa fa-plus" ] [] ]
        ]


viewProjectList : List Project -> Html Msg
viewProjectList projects =
    let
        latestProjects =
            sortByDatetime .updatedAt projects
    in
        div []
            [ h6 [] [ text "Projects" ]
            , ul [ class "list-group list-group-flush" ] (List.map viewProjectListItem latestProjects)
            ]


projectFormConfig : ProjectForm.Config Msg
projectFormConfig =
    { setNameMsg = SetProjectFormName
    , setRepositoryMsg = SetProjectFormRepository
    , setPrivateKeyMsg = SetProjectFormPrivateKey
    , submitMsg = SubmitProjectForm
    }


knownHostFormConfig : KnownHostForm.Config Msg
knownHostFormConfig =
    { setScannedKeyMsg = SetKnownHostFormScannedKey
    , submitMsg = SubmitKnownHostForm
    }


viewNewProjectModal : Model -> Html Msg
viewNewProjectModal { newProjectForm, newKnownHostForm, newProjectModalVisibility, knownHosts } =
    Modal.config CloseNewProjectModal
        |> Modal.withAnimation AnimateNewProjectModal
        |> Modal.large
        |> Modal.hideOnBackdropClick (not newProjectForm.submitting)
        |> Modal.h3 [] [ text "Create project" ]
        |> Modal.body [] [ viewCombinedForm knownHosts newProjectForm newKnownHostForm ]
        |> Modal.footer [] [ ProjectForm.viewSubmitButton projectFormConfig newProjectForm ]
        |> Modal.view newProjectModalVisibility


viewCombinedForm : List KnownHost -> ProjectForm.Context -> KnownHostForm.Context -> Html Msg
viewCombinedForm knownHosts projectForm knownHostForm =
    let
        projectFormView =
            ProjectForm.view projectFormConfig projectForm

        knownHostFormView =
            if ProjectForm.isUnknownKnownHost projectForm.form.repository.value knownHosts then
                KnownHostForm.view knownHostFormConfig knownHostForm
            else
                text ""
    in
        div []
            [ projectFormView
            , knownHostFormView
            ]


viewProjectListItem : Project -> Html Msg
viewProjectListItem project =
    let
        route =
            Route.Project project.slug ProjectRoute.Overview

        lastUpdatedText =
            "Last updated " ++ formatDate (DateTime.date project.updatedAt)

        smallText =
            project.repository

        projectLink =
            a
                [ Route.href route
                , onClickPage NewUrl route
                ]
                [ text project.name ]
    in
        li [ class "list-group-item flex-column align-items-start px-0" ]
            [ div [ class "d-flex w-100 justify-content-between" ]
                [ h5 [ class "mb-1" ] [ projectLink ]
                , small [] [ text lastUpdatedText ]
                ]
            , small [] [ text smallText ]
            ]



-- UPDATE --


type Msg
    = NoOp_
    | NewUrl String
    | AddProject Encode.Value
    | AddKnownHost Encode.Value
    | CloseNewProjectModal
    | AnimateNewProjectModal Modal.Visibility
    | ShowNewProjectModal
    | SubmitProjectForm
    | SetProjectFormName String
    | SetProjectFormRepository String
    | SetProjectFormPrivateKey String
    | SetKnownHostFormScannedKey String
    | SubmitKnownHostForm
    | ProjectCreated (Result Request.Errors.HttpError Project)
    | KnownHostCreated (Result Request.Errors.HttpError KnownHost)
    | SetGitUrl (Maybe GitUrl)


type ExternalMsg
    = NoOp
    | HandleRequestError Request.Errors.HandledError


formError : String -> List ( String, String )
formError errorMsg =
    [ "" => errorMsg ]


update : Context -> Session msg -> Msg -> Model -> ( ( Model, Cmd Msg ), ExternalMsg )
update context session msg model =
    case msg of
        NoOp_ ->
            model
                => Cmd.none
                => NoOp

        NewUrl url ->
            model
                => newUrl url
                => NoOp

        SetGitUrl maybeGitUrl ->
            { model | newProjectForm = ProjectForm.updateGitUrl maybeGitUrl model.newProjectForm }
                => Cmd.none
                => NoOp

        CloseNewProjectModal ->
            { model
                | newProjectModalVisibility = Modal.hidden
                , newProjectForm = ProjectForm.init
            }
                => Cmd.none
                => NoOp

        AnimateNewProjectModal visibility ->
            { model | newProjectModalVisibility = visibility }
                => Cmd.none
                => NoOp

        SetProjectFormName name ->
            { model | newProjectForm = ProjectForm.update model.newProjectForm ProjectForm.Name name }
                => Cmd.none
                => NoOp

        SetProjectFormRepository repository ->
            { model | newProjectForm = ProjectForm.update model.newProjectForm ProjectForm.Repository repository }
                => Ports.parseGitUrl repository
                => NoOp

        SetProjectFormPrivateKey privateKey ->
            { model | newProjectForm = ProjectForm.update model.newProjectForm ProjectForm.PrivateKey privateKey }
                => Cmd.none
                => NoOp

        SubmitProjectForm ->
            let
                cmdFromAuth authToken =
                    authToken
                        |> Request.Project.create context (ProjectForm.submitValues model.newProjectForm)
                        |> Task.attempt ProjectCreated

                cmd =
                    session
                        |> Session.attempt "create project" cmdFromAuth
                        |> Tuple.second
            in
                { model | newProjectForm = Form.submit model.newProjectForm }
                    => cmd
                    => NoOp

        SubmitKnownHostForm ->
            let
                cmdFromAuth authToken =
                    authToken
                        |> Request.KnownHost.create context (KnownHostForm.submitValues model.newKnownHostForm)
                        |> Task.attempt KnownHostCreated

                cmd =
                    session
                        |> Session.attempt "create known host" cmdFromAuth
                        |> Tuple.second
            in
                { model | newKnownHostForm = Form.submit model.newKnownHostForm }
                    => cmd
                    => NoOp

        SetKnownHostFormScannedKey key ->
            { model | newKnownHostForm = KnownHostForm.update model.newKnownHostForm KnownHostForm.ScannedKey key }
                => Cmd.none
                => NoOp

        ProjectCreated (Err err) ->
            let
                formErrors =
                    ProjectForm.serverErrorToFormError
                        |> Form.updateServerErrors (formError "Unable to process project.")

                ( updatedProjectForm, externalMsg ) =
                    case err of
                        Request.Errors.HandledError handledError ->
                            model.newProjectForm
                                => HandleRequestError handledError

                        Request.Errors.UnhandledError (Http.BadStatus response) ->
                            let
                                errors =
                                    response.body
                                        |> decodeString ProjectForm.errorsDecoder
                                        |> Result.withDefault []
                            in
                                model.newProjectForm
                                    |> Form.updateServerErrors errors ProjectForm.serverErrorToFormError
                                    => NoOp

                        _ ->
                            model.newProjectForm
                                |> formErrors
                                => NoOp
            in
                { model | newProjectForm = Form.submitting False updatedProjectForm }
                    => Cmd.none
                    => externalMsg

        ProjectCreated (Ok project) ->
            { model
                | newProjectForm = ProjectForm.init
                , newProjectModalVisibility = Modal.hidden
                , projects = addProject model.projects project
            }
                => Cmd.none
                => NoOp

        KnownHostCreated (Err err) ->
            let
                formErrors =
                    KnownHostForm.serverErrorToFormError
                        |> Form.updateServerErrors (formError "Unable to process known host.")

                ( updatedKnownHostForm, externalMsg ) =
                    case err of
                        Request.Errors.HandledError handledError ->
                            model.newKnownHostForm
                                => HandleRequestError handledError

                        Request.Errors.UnhandledError (Http.BadStatus response) ->
                            let
                                errors =
                                    response.body
                                        |> decodeString KnownHostForm.errorsDecoder
                                        |> Result.withDefault []
                            in
                                model.newKnownHostForm
                                    |> Form.updateServerErrors errors KnownHostForm.serverErrorToFormError
                                    => NoOp

                        _ ->
                            model.newKnownHostForm
                                |> formErrors
                                => NoOp
            in
                { model | newKnownHostForm = Form.submitting False updatedKnownHostForm }
                    => Cmd.none
                    => externalMsg

        KnownHostCreated (Ok knownHost) ->
            { model
                | newKnownHostForm = KnownHostForm.init
                , knownHosts = addKnownHost model.knownHosts knownHost
            }
                => Cmd.none
                => NoOp

        ShowNewProjectModal ->
            { model | newProjectModalVisibility = Modal.shown }
                => Task.attempt (always NoOp_) (Dom.focus "name")
                => NoOp

        AddProject projectJson ->
            let
                projects =
                    case Decode.decodeValue Project.decoder projectJson of
                        Ok project ->
                            addProject model.projects project

                        Err _ ->
                            model.projects
            in
                { model | projects = projects }
                    => Cmd.none
                    => NoOp

        AddKnownHost knownHostJson ->
            let
                knownHosts =
                    case Decode.decodeValue KnownHost.decoder knownHostJson of
                        Ok knownHost ->
                            addKnownHost model.knownHosts knownHost

                        Err _ ->
                            model.knownHosts
            in
                { model | knownHosts = knownHosts }
                    => Cmd.none
                    => NoOp
