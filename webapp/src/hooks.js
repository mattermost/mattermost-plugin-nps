// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {General} from 'mattermost-redux/constants';
import {makeGetChannel} from 'mattermost-redux/selectors/entities/channels';
import {getUser} from 'mattermost-redux/selectors/entities/users';

import * as Actions from './actions';

export default class Hooks {
    constructor(store) {
        this.store = store;
    }

    messageWillBePosted = (post) => {
        const channel = makeGetChannel()(this.store.getState(), {id: post.channel_id});

        if (channel.type !== General.DM_CHANNEL) {
            return Promise.resolve({post});
        }

        // makeGetChannel passes the channel through completeDirectChannelInfo, so it has extra data that helps here
        const teammate = getUser(this.store.getState(), channel.teammate_id);

        if (teammate.username !== 'feedbackbot') {
            return Promise.resolve({post});
        }

        return new Promise((resolve) => {
            const onConfirm = (email) => {
                post.props.feedback_email = email;
                resolve({post});
            };
            const onCancel = () => resolve({error: {message: 'Feedback not sent.'}});

            this.store.dispatch(Actions.showConfirmationModal(onConfirm, onCancel));
        });
    }
}
