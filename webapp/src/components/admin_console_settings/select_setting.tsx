import React from 'react';

import BaseSetting from './base_setting';
import ReactSelect, { ActionMeta } from 'react-select';

export type OptionType = {
    label: string | JSX.Element;
    value: string;
}

type Props = {
    id: string;
    name: string;
    helpText: string;
    onChange: (value: string) => void;
    value: string;
    getOptions: () => OptionType[];
    disabled?: boolean;
};

const SelectSetting = (props: Props) => {
    return (
        <BaseSetting
            {...props}
        >
            <ReactSelect
                id={props.id}
                onChange={(v) => {
                    console.log("ReactSelect", v)
                    props.onChange((v as OptionType).value);
                }}
                isDisabled={props.disabled}
                value={props.value}
                hideSelectedOptions={true}
                isSearchable={true}
                options={props.getOptions()}
            />
        </BaseSetting>
    );
};

export default SelectSetting;
