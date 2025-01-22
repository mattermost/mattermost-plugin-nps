// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {getChannel} from 'mattermost-redux/selectors/entities/channels';
import {getCurrentRelativeTeamUrl} from 'mattermost-redux/selectors/entities/teams';
import {getCurrentUserId} from 'mattermost-redux/selectors/entities/users';

import {navigateToChannel} from './browser_routing';
import * as ActionTypes from './action_types';

export function connected(client) {
    return (dispatch, getState) => {
        const currentUserId = getCurrentUserId(getState());

        if (currentUserId) {
            client.connected();
        }
    };
}

export function userWantsToGiveFeedback(client) {
    return (_, getState) => {
        const currentUserId = getCurrentUserId(getState());
        if (!currentUserId) {
            return;
        }

        client.userWantsToGiveFeedback().then(({data}) => {
            const channel = getChannel(getState(), data.channel_id);
            navigateToChannel(getCurrentRelativeTeamUrl(getState()), channel.name);
        });
    };
}

export function showConfirmationModal(onConfirm, onCancel) {
    return {
        type: ActionTypes.SHOW_CONFIRMATION_MODAL,
        onCancel,
        onConfirm,
    };
}

export function hideConfirmationModal() {
    return {
        type: ActionTypes.HIDE_CONFIRMATION_MODAL,
    };
}

export function windowResized(windowWidth) {
    return {
        type: ActionTypes.WINDOW_RESIZED,
        windowWidth,
    };
}
