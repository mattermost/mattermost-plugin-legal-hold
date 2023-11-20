import {LegalHold} from "@/types";
import React from "react";
import {UserProfile} from "mattermost-redux/types/users";


interface LegalHoldRowProps {
    legalHold: LegalHold;
    users: UserProfile[];
    releaseLegalHold: Function;
}

const LegalHoldRow = (props: LegalHoldRowProps) => {
    const lh = props.legalHold;
    const startsAt = (new Date(lh.starts_at)).toLocaleDateString();
    const endsAt = (new Date(lh.ends_at)).toLocaleDateString();

    const release = () => {
        props.releaseLegalHold(lh.id);
    };

    const usernames = props.users.map((user) => {
       if (user) {
           return `@${user.username} `;
       } else {
           return `loading...`;
       }
    });

    return (
        <React.Fragment>
            <div>{lh.display_name}</div>
            <div>{lh.name}</div>
            <div>{startsAt}</div>
            <div>{endsAt}</div>
            <div>{usernames}</div>
            <div><a href="#">Edit</a></div>
            <div><a href="#" onClick={release}>Release</a></div>
        </React.Fragment>
    );
}

export default LegalHoldRow;
