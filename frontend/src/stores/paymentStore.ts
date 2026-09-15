import {reactive}from'vue';import type{Payment}from'../types';export const paymentStore=reactive<{items:Payment[]}>({items:[]});
