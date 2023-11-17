import {connect} from 'react-redux';
import {AnyAction, bindActionCreators, Dispatch} from 'redux';

import {
    getProfiles,
    searchProfiles as reduxSearchProfiles
} from 'mattermost-redux/actions/users';

import UsersInput from './users_input.jsx';

const searchProfiles = (term: string, options = {}) => {
    if (!term) {
        return getProfiles(0, 20, options);
    }
    return reduxSearchProfiles(term, options);
};

function mapDispatchToProps(dispatch: Dispatch<AnyAction>) {
    return {
        actions: bindActionCreators({
            searchProfiles,
        }, dispatch),
    };
}

export default connect(null, mapDispatchToProps)(UsersInput);
