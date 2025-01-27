// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import * as Actions from './actions';
import {Client} from './client';
import {POST_NPS_SURVEY} from './constants';
import Hooks from './hooks';
import manifest from './manifest';
import reducer from './reducers';
import {useSurveyPost as shouldUseSurveyPost} from './selectors';

import Root from './components/root';
import SurveyPost from './components/survey_post';

export default class Plugin {
    constructor() {
        this.registry = null;

        this.store = null;
        this.unsubscribe = null;

        this.client = null;

        this.overrideSurveyPost = false;
        this.surveyPostComponentId = '';
    }

    onStateChange = () => {
        this.registerSurveyPost(this.store.getState());
    }

    onWindowResize = () => {
        this.store.dispatch(Actions.windowResized(window.innerWidth));
    }

    registerSurveyPost = (state) => {
        const overrideSurveyPost = shouldUseSurveyPost(state);

        // this.overrideSurveyPost has to be updated first since registerPostTypeComponent calls this again

        if (overrideSurveyPost && !this.overrideSurveyPost) {
            // There's enough space to use the custom survey post, so register it
            this.overrideSurveyPost = true;
            this.surveyPostComponentId = this.registry.registerPostTypeComponent(POST_NPS_SURVEY, SurveyPost);
        } else if (!overrideSurveyPost && this.overrideSurveyPost) {
            // There's not enough space to use the custom score post, so remove it
            this.overrideSurveyPost = false;

            this.registry.unregisterPostTypeComponent(this.surveyPostComponentId);
            this.surveyPostComponentId = '';
        }
    }

    onGiveFeedbackClick = () => {
        this.store.dispatch(Actions.userWantsToGiveFeedback(this.client));
    }

    initialize(registry, store) {
        this.registry = registry;

        this.store = store;
        this.unsubscribe = store.subscribe(this.onStateChange);

        // Register reducer
        registry.registerReducer(reducer);

        // Register hooks
        const hooks = new Hooks(store);
        registry.registerMessageWillBePostedHook(hooks.messageWillBePosted);

        // Register components
        registry.registerRootComponent(Root);
        registry.registerUserGuideDropdownMenuAction('Give feedback', this.onGiveFeedbackClick);

        window.addEventListener('resize', this.onWindowResize);
        this.onWindowResize();

        // Initialize client
        this.client = new Client();
        store.dispatch(Actions.connected(this.client));
    }

    uninitialize() {
        window.removeEventListener('resize', this.onWindowResize);

        if (this.unsubscribe) {
            this.unsubscribe();
        }
    }
}

window.registerPlugin(manifest.id, new Plugin());
