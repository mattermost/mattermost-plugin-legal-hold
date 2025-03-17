import {connect} from 'react-redux';

import {GlobalState} from 'mattermost-redux/types/store';
import {getUser} from 'mattermost-redux/selectors/entities/users';
import {getGroup} from 'mattermost-redux/selectors/entities/groups';

import LegalHoldRow from '@/components/legal_hold_table/legal_hold_row/legal_hold_row';
import {LegalHold} from '@/types';

type OwnProps = {
    legalHold: LegalHold;
}

function makeMapStateToProps() {
    return (state: GlobalState, ownProps: OwnProps) => {
        if (ownProps.legalHold.user_ids === null) {
            return {
                groups: [],
                users: [],
            };
        }

        const groups = ownProps.legalHold.group_ids.map((group_id) => getGroup(state, group_id));
        const users = ownProps.legalHold.user_ids.map((user_id) => getUser(state, user_id));
        return {
            groups,
            users,
        };
    };
}

export default connect(makeMapStateToProps)(LegalHoldRow);
