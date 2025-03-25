import {connect} from 'react-redux';
import {AnyAction, bindActionCreators, Dispatch} from 'redux';

import Client from '@/client';
import GroupsInput from './groups_input.jsx';

// Function to search groups via the plugin API
const searchGroups = (term: string) => {
    return async () => {
        try {
            return await Client.searchGroups(term);
        } catch (error) {
            console.log(error); //eslint-disable-line no-console
            throw error;
        }
    };
};

function mapDispatchToProps(dispatch: Dispatch<AnyAction>) {
    return {
        actions: bindActionCreators({
            searchGroups,
        }, dispatch),
    };
}

export default connect(null, mapDispatchToProps)(GroupsInput);
