import {connect} from 'react-redux';

import {doPostActionWithCookie} from 'mattermost-redux/actions/posts';

import SurveyPost from './survey_post';

const mapDispatchToProps = {
    doPostActionWithCookie,
};

export default connect(null, mapDispatchToProps)(SurveyPost);
