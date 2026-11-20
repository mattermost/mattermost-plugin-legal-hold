import {shallowEqual, useSelector} from 'react-redux';

import type {Group} from '@mattermost/types/groups';
import type {GlobalState} from '@mattermost/types/store';
import type {UserProfile} from '@mattermost/types/users';

import {getGroup} from 'mattermost-redux/selectors/entities/groups';
import {getUser} from 'mattermost-redux/selectors/entities/users';

import type {LegalHold} from '@/types';

// Returns the users and groups that a legal hold targets, as far as they are loaded in the store.
// `complete` is true only when the store holds every targeted user and group.
export const useLegalHoldEntities = (legalHold: LegalHold | null) => {
    const users = useSelector((state: GlobalState) => {
        return (legalHold?.user_ids ?? []).
            map((id) => getUser(state, id)).
            filter((user): user is UserProfile => Boolean(user));
    }, shallowEqual);

    const groups = useSelector((state: GlobalState) => {
        return (legalHold?.group_ids ?? []).
            map((id) => getGroup(state, id)).
            filter((group): group is Group => Boolean(group));
    }, shallowEqual);

    const complete = users.length === (legalHold?.user_ids?.length ?? 0) &&
        groups.length === (legalHold?.group_ids?.length ?? 0);

    return {users, groups, complete};
};
